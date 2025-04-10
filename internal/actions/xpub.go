package actions

import (
	"bytes"
	"errors"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
)

func getAddressFromZpub(key *hdkeychain.ExtendedKey, external bool, index int) (string, error) {
	changeIndex := uint32(0)

	if !external {
		changeIndex = uint32(1)
	}

	changeKey, err := key.Derive(changeIndex)
	if err != nil {
		return "", err
	}

	addressKey, err := changeKey.Derive(uint32(index))
	if err != nil {
		return "", err
	}

	pk, err := addressKey.ECPubKey()
	if err != nil {
		return "", err
	}

	witness := btcutil.Hash160(pk.SerializeCompressed())
	awpkh, err := btcutil.NewAddressWitnessPubKeyHash(witness, &chaincfg.MainNetParams)
	if err != nil {
		return "", err
	}

	return awpkh.EncodeAddress(), nil
}

func GetAddressFromExtPubKey(extPubKey string, external bool, addressIndex int) (string, error) {
	key, err := hdkeychain.NewKeyFromString(extPubKey)
	if err != nil {
		return "", err
	}

	// zpub support only
	if bytes.Equal(key.Version(), []byte{0x04, 0xb2, 0x47, 0x46}) == false {
		return "", errors.New("unsupported xpub")
	}

	return getAddressFromZpub(key, external, addressIndex)
}
