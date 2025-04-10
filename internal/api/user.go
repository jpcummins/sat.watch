package api

import (
	"context"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
)

type User struct {
	Model
	api          *API
	Username     string
	PasswordHash string `db:"password_hash"`
}

func (api *API) CreateUser() (User, error) {
	now := time.Now()
	userId := uuid.New().String()
	user := User{
		Model: Model{
			ID:        userId,
			CreatedAt: &now,
			UpdatedAt: &now,
		},
		api:      api,
		Username: userId,
	}
	err := api.createUser(user)
	return user, err
}

func (api *API) GetUser(userId string) (User, error) {
	var user User

	sql := `
		SELECT 
		    id, 
		    created_at, 
		    updated_at, 
		    username,
		    password_hash
		FROM 
		    users 
		WHERE 
		    id = $1 
		    AND deleted_at IS NULL;`

	err := pgxscan.Get(context.TODO(), api.db, &user, sql, userId)
	user.api = api
	return user, err
}

func (api *API) GetUserByUsername(username string) (User, error) {
	var user User

	sql := `
		SELECT 
		    id, 
		    created_at, 
		    updated_at, 
		    username,
		    password_hash
		FROM 
		    users 
		WHERE 
		    username = $1 
		    AND deleted_at IS NULL;`

	err := pgxscan.Get(context.TODO(), api.db, &user, sql, username)
	user.api = api
	return user, err
}

func (api *API) SoftDeleteUser(userId string) error {
	sql := `
		UPDATE users
		SET deleted_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	_, err := api.db.Exec(context.TODO(), sql, userId)
	return err
}

func (api *API) createUser(user User) error {
	sql := `
		INSERT INTO users (
		    id, 
		    created_at, 
		    updated_at, 
		    username
		) 
		VALUES (
		    $1, 
		    $2, 
		    $3, 
		    $4
		);`

	_, err := api.db.Exec(context.TODO(), sql, user.ID, user.CreatedAt, user.UpdatedAt, user.Username)
	return err
}
