package api

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
)

type Email struct {
	Model
	UserID              string     `db:"user_id"`
	Description         string     `form:"description"`
	Email               string     `form:"email"`
	IsVerified          bool       `db:"is_verified"`
	VerificationCode    string     `db:"verification_code"`
	VerificationExpires *time.Time `db:"verification_expires"`
	VerifiedOn          *time.Time `db:"verified_on"`
	Pubkey              *string    `db:"pgp_pubkey"`
}

const verificationExpiry = time.Hour * 24

func (api *API) CreateEmail(userId string, email string, description string, pubkey string) error {
	pubkeyTrimmed := strings.TrimSpace(pubkey)
	if pubkeyTrimmed != "" {
		isValid := isPubkeyValid(pubkeyTrimmed)

		if !isValid {
			return errors.New("Invalid Pubkey")
		}
	}

	expires := time.Now().Add(verificationExpiry)
	_, err := api.db.Exec(context.Background(), "INSERT INTO emails (user_id, email, description, is_verified, verification_expires, pgp_pubkey) VALUES ($1, $2, $3, false, $4, $5)", userId, email, description, expires, pubkeyTrimmed)
	return err
}

func (api *API) ResetVerificationCode(userId string, notificationId string) (Email, error) {
	email, err := api.GetEmail(userId, notificationId)
	if err != nil {
		return email, err
	}

	if email.IsVerified {
		return email, errors.New("Email already verified.")
	}

	expires := time.Now().Add(verificationExpiry)
	uuid := uuid.New()
	email.IsVerified = false
	email.VerificationCode = uuid.String()
	email.VerificationExpires = &expires
	email.VerifiedOn = nil

	_, err = api.db.Exec(context.Background(), "UPDATE emails SET is_verified = $1, verification_expires = $2, verification_code = $3, verified_on = $4 WHERE user_id = $5 AND id = $6 AND deleted_at IS NULL", email.IsVerified, email.VerificationExpires, email.VerificationCode, email.VerifiedOn, userId, notificationId)
	return email, err
}

func (api *API) VerifyEmail(userId string, notificationId string, verificationCode string) error {
	email, err := api.GetEmail(userId, notificationId)
	if err != nil {
		return err
	}

	if email.IsVerified {
		return nil
	}

	if email.VerificationCode != verificationCode {
		return errors.New("invalid verification code")
	}

	if email.VerificationExpires.Before(time.Now()) {
		return errors.New("expired verification code")
	}

	verifiedOn := time.Now()

	_, err = api.db.Exec(context.Background(), "UPDATE emails SET is_verified = $1, verified_on = $2 WHERE user_id = $3 AND id = $4 AND deleted_at IS NULL", true, &verifiedOn, userId, notificationId)
	return err
}

func (api *API) GetEmail(userId string, notificationId string) (Email, error) {
	var email Email
	err := pgxscan.Get(context.Background(), api.db, &email, "SELECT id, created_at, updated_at, user_id, email, description, is_verified, verification_code, verification_expires, verified_on, pgp_pubkey FROM emails WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL", userId, notificationId)
	return email, err
}

func (api *API) GetEmailByAddress(userId string, address string) (Email, error) {
	var email Email
	err := pgxscan.Get(context.Background(), api.db, &email, "SELECT id, created_at, updated_at, user_id, email, description, is_verified, verification_code, verification_expires, verified_on, pgp_pubkey FROM emails WHERE user_id = $1 AND email = $2 AND deleted_at IS NULL", userId, address)
	return email, err
}

func (api *API) GetVerifiedUserEmails(userId string) ([]Email, error) {
	var emails []Email
	err := pgxscan.Select(context.Background(), api.db, &emails, "SELECT id, created_at, updated_at, user_id, email, description, is_verified, verification_code, verification_expires, verified_on, pgp_pubkey FROM emails WHERE user_id = $1 AND deleted_at IS NULL AND is_verified = true", userId)
	return emails, err
}

func (api *API) GetUserEmails(userId string) ([]Email, error) {
	var emails []Email
	err := pgxscan.Select(context.Background(), api.db, &emails, "SELECT id, created_at, updated_at, user_id, email, description, is_verified, verification_code, verification_expires, verified_on, pgp_pubkey FROM emails WHERE user_id = $1 AND deleted_at IS NULL", userId)
	return emails, err
}

func (api *API) UpdateEmailDescription(userId string, notificationId string, description string) error {
	_, err := api.db.Exec(context.Background(), "UPDATE emails SET description = $1 WHERE user_id = $2 AND id = $3 AND deleted_at IS NULL", description, userId, notificationId)
	return err
}

func (api *API) UpdateEmailPubkey(userId string, notificationId string, pubkey string) error {
	pubkeyTrimmed := strings.TrimSpace(pubkey)
	isValid := isPubkeyValid(pubkey)

	if !isValid {
		return errors.New("Invalid Pubkkey")
	}

	_, err := api.db.Exec(context.Background(), "UPDATE emails SET pgp_pubkey = $1 WHERE user_id = $2 AND id = $3 AND deleted_at IS NULL", pubkeyTrimmed, userId, notificationId)
	return err
}

func (api *API) DeleteEmail(userId string, notificationId string) error {
	_, err := api.db.Exec(context.Background(), "DELETE FROM emails WHERE user_id = $1 AND id = $2", userId, notificationId)
	if err != nil {
		return err
	}
	return nil
}

func (api *API) DeleteEmails(userId string) error {
	sql := `
		DELETE FROM emails
		WHERE user_id = $1`

	_, err := api.db.Exec(context.TODO(), sql, userId)
	return err
}

func (api *API) LogAlertEmail(email string, user_id string, address_id string, transaction_id string, emailError error) error {
	sql := `
		INSERT INTO email_log (email, user_id, address_id, transaction_id, error)
		VALUES (
		  $1, 
		  $2, 
		  $3,
		  $4,
		  $5
		)`

	var errorString string
	if emailError != nil {
		errorString = emailError.Error()
	}

	_, err := api.db.Exec(context.TODO(), sql, email, user_id, address_id, transaction_id, errorString)
	return err
}

func (api *API) GetDailyEmailCount(address_id string, user_id string) (int, error) {
	sql := `
		SELECT COUNT(*)
		FROM email_log
		WHERE user_id = $1
		  AND address_id = $2
		  AND created_at >= now() - INTERVAL '24 hours'
		  AND error = '';
	`
	var count int
	err := pgxscan.Get(context.Background(), api.db, &count, sql, user_id, address_id)
	return count, err
}

func isPubkeyValid(pubkey string) bool {
	_, err := crypto.NewKeyFromArmored(pubkey)
	return err == nil
}
