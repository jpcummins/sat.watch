package api

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jpcummins/go-electrum/electrum"
)

type Address struct {
	Model
	UserID       string  `db:"user_id"`
	XpubID       *string `db:"xpub_id"`
	Address      string  `form:"address"`
	Scripthash   string  `db:"scripthash"`
	Name         *string `form:"name"`
	IsExternal   bool    `db:"is_external"`
	AddressIndex int     `db:"address_index"`
	UTXOs        []*electrum.ListUnspentResult
}

func (api *API) CreateAddress(addresses ...Address) error {
	batch := &pgx.Batch{}
	for i, address := range addresses {
		scripthash, err := electrum.AddressToElectrumScriptHash(address.Address)
		if err != nil {
			return err
		}
		addresses[i].Scripthash = scripthash

		batch.Queue("INSERT INTO addresses (id, user_id, xpub_id, address, scripthash, name, is_external, address_index) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
			address.ID,
			address.UserID,
			address.XpubID,
			address.Address,
			scripthash,
			address.Name,
			address.IsExternal,
			address.AddressIndex)

		api.monitor.EnqueueScan(scripthash)
	}

	err := api.db.SendBatch(context.Background(), batch).Close()
	if err != nil {
		return errors.Join(err, errors.New(fmt.Sprintf("Unable to save addresses: %+v", addresses)))
	}

	api.mu.Lock()
	defer api.mu.Unlock()
	api.addresses = append(api.addresses, addresses...)
	return nil
}

func (api *API) UpdateAddressUTXOs(scriptHash string, utxos []*electrum.ListUnspentResult) error {
	api.mu.Lock()
	defer api.mu.Unlock()

	amount := uint64(0)
	for _, utxo := range utxos {
		amount = amount + utxo.Value
	}

	for i, address := range api.addresses {
		if scriptHash == address.Scripthash {
			api.addresses[i].UTXOs = utxos
		}
	}

	return nil
}

func (api *API) GetAddress(address string) (*Address, error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	for _, watchedAddress := range api.addresses {
		if address == watchedAddress.Address {
			return &watchedAddress, nil
		}
	}
	return nil, nil
}

func (api *API) GetAddressById(addressId string, userId string) (*Address, error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	for _, watchedAddress := range api.addresses {
		if addressId == watchedAddress.ID && userId == watchedAddress.UserID {
			return &watchedAddress, nil
		}
	}
	return nil, nil
}

func (api *API) GetAddresses() []Address {
	api.mu.Lock()
	defer api.mu.Unlock()

	addresses := make([]Address, len(api.addresses))
	copy(addresses, api.addresses)
	return addresses
}

func (api *API) GetAddressesForXpub(userId string, xpubId string) []Address {
	api.mu.Lock()
	defer api.mu.Unlock()

	var addresses []Address
	for _, address := range api.addresses {
		if address.UserID == userId && address.XpubID != nil && *address.XpubID == xpubId {
			addresses = append(addresses, address)
		}
	}
	return addresses
}

func (api *API) GetAddressesWithoutXpub(userId string) []Address {
	api.mu.Lock()
	defer api.mu.Unlock()

	var addresses []Address
	for _, address := range api.addresses {
		if address.XpubID == nil && address.UserID == userId {
			addresses = append(addresses, address)
		}
	}
	return addresses
}

func (api *API) GetAddressesForUser(userId string) []Address {
	api.mu.Lock()
	defer api.mu.Unlock()

	var addresses []Address
	for _, address := range api.addresses {
		if address.UserID == userId {
			addresses = append(addresses, address)
		}
	}

	slices.SortFunc(addresses, func(a Address, b Address) int {
		if a.CreatedAt == nil || b.CreatedAt == nil {
			return 0
		}
		return (*a.CreatedAt).Compare(*b.CreatedAt)
	})

	return addresses
}

func (api *API) DeleteAddresses(userId string) error {
	sql := `
		UPDATE addresses
		   SET deleted_at = NOW(),
		       address = '',
		       scripthash = ''
		 WHERE user_id = $1`
	_, err := api.db.Exec(context.TODO(), sql, userId)
	return err
}

func (api *API) DeleteAddress(userId string, addressId string) error {
	sql := `
		UPDATE addresses
		   SET deleted_at = NOW(),
		       address = '',
		       scripthash = ''
		 WHERE id = $1 AND user_id = $2`
	_, err := api.db.Exec(context.TODO(), sql, addressId, userId)
	if err != nil {
		return err
	}

	api.mu.Lock()
	defer api.mu.Unlock()

	api.addresses = slices.DeleteFunc(api.addresses, func(address Address) bool {
		if address.ID == addressId && address.UserID == userId {
			return true
		}
		return false
	})

	return err
}
