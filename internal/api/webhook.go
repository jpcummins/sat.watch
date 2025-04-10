package api

import (
	"context"
	"github.com/georgysavva/scany/v2/pgxscan"
)

type Webhook struct {
	Model
	UserID string `db:"user_id"`
	Name   string `form:"name" binding:"required"`
	Url    string `form:"url" binding:"required,http_url,startswith=https"`
}

func (api *API) CreateWebhook(userId string, name string, url string) error {
	_, err := api.db.Exec(context.Background(), "INSERT INTO webhooks (user_id, name, url) VALUES ($1, $2, $3)", userId, name, url)
	return err
}

func (api *API) GetWebhook(userId string, id string) (Webhook, error) {
	var webhook Webhook
	err := pgxscan.Get(context.Background(), api.db, &webhook, "SELECT id, created_at, updated_at, user_id, name, url FROM webhooks WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL", userId, id)
	return webhook, err
}

func (api *API) GetUserWebhooks(userId string) ([]Webhook, error) {
	var webhooks []Webhook
	err := pgxscan.Select(context.Background(), api.db, &webhooks, "SELECT id, created_at, updated_at, user_id, name, url FROM webhooks WHERE user_id = $1 AND deleted_at IS NULL", userId)
	return webhooks, err
}

func (api *API) GetWebhooks() ([]Webhook, error) {
	var webhooks []Webhook
	err := pgxscan.Select(context.Background(), api.db, &webhooks, "SELECT id, created_at, updated_at, user_id, name, url FROM webhooks WHERE deleted_at IS NULL")
	return webhooks, err
}

func (api *API) UpdateWebhook(userId string, notificationId string, url string, name string) error {
	_, err := api.db.Exec(context.Background(), "UPDATE webhooks SET url = $1, name = $2 WHERE user_id = $3 AND id = $4 AND deleted_at IS NULL", url, name, userId, notificationId)
	return err
}

func (api *API) DeleteWebhook(userId string, notificationId string) error {
	_, err := api.db.Exec(context.Background(), "DELETE FROM webhooks WHERE user_id = $1 AND id = $2", userId, notificationId)
	return err
}

func (api *API) DeleteWebhooks(userId string) error {
	sql := `
		DELETE FROM webhooks
		WHERE user_id = $1`

	_, err := api.db.Exec(context.TODO(), sql, userId)
	return err
}
