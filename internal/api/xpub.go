package api

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
)

type Xpub struct {
	Model
	UserID string  `db:"user_id"`
	Pubkey string  `form:"pubkey" binding:"required"`
	Name   *string `form:"name"`
	Gap    int
}

func (api *API) CreateXpub(userId string, pubkey string, name *string, gap int) (Xpub, error) {
	var id string
	err := api.db.QueryRow(context.Background(), "INSERT INTO xpubs (user_id, pubkey, name, gap) VALUES ($1, $2, $3, $4) RETURNING id", userId, pubkey, name, gap).Scan(&id)
	return Xpub{Model{ID: id}, userId, pubkey, name, gap}, err
}

func (api *API) GetXpubs(userId string) ([]Xpub, error) {
	var xpubs []Xpub
	err := pgxscan.Select(context.Background(), api.db, &xpubs, "SELECT id, created_at, updated_at, pubkey, name, gap FROM xpubs WHERE user_id = $1 AND deleted_at IS NULL", userId)
	return xpubs, err
}

func (api *API) GetXpub(userId string, xpub_id string) (Xpub, error) {
	var xpub Xpub
	err := pgxscan.Get(context.Background(), api.db, &xpub, "SELECT id, created_at, updated_at, user_id, pubkey, name, gap FROM xpubs WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL", xpub_id, userId)
	return xpub, err
}

func (api *API) DeleteUserXpubs(userId string) error {
	sql := `
		DELETE FROM xpubs
		WHERE user_id = $1`

	_, err := api.db.Exec(context.TODO(), sql, userId)
	return err
}

func (api *API) DeleteXpub(userId string, xpubId string) error {
	addresses := api.GetAddressesForXpub(userId, xpubId)

	for _, address := range addresses {
		err := api.DeleteAddress(userId, address.ID)
		if err != nil {
			return err
		}
	}

	sql := `
		UPDATE xpubs SET pubkey = '', name = '', deleted_at = NOW()
		WHERE user_id = $1 AND id = $2`

	_, err := api.db.Exec(context.TODO(), sql, userId, xpubId)
	return err
}
