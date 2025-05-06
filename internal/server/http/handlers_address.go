package http

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jpcummins/go-electrum/electrum"
	"github.com/jpcummins/satwatch/internal/api"
	"github.com/jpcummins/satwatch/internal/clients"
	"github.com/jpcummins/satwatch/internal/configs"
	"github.com/jpcummins/satwatch/internal/lib"
	"github.com/jpcummins/satwatch/internal/server/http/web/templates"
	"github.com/labstack/echo/v4"
)

type AddressController struct {
	API            *api.API
	Gap            int
	EmailClient    EmailClient
	ElectrumClient *electrum.Client
	BitcoinClient  clients.BitcoinClient
	Config         *configs.Config
}

func (ac AddressController) Index(c echo.Context) error {
	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	address, err := ac.API.GetAddressById(c.Param("address"), user.Model.ID)
	if err != nil {
		logger(c).Error().Err(err).Msg("GetAddressById lookup failed")
		return Render(c, http.StatusInternalServerError, templates.PageAddressNotFound())
	}

	unspent, err := ac.ElectrumClient.ListUnspent(context.TODO(), address.Scripthash)
	if err != nil {
		logger(c).Error().Str("scripthash", address.Scripthash).Err(err).Msg("ListUnspent failed")
		return Render(c, http.StatusInternalServerError, templates.PageAddressNotFound())
	}

	balances, err := ac.ElectrumClient.GetBalance(context.TODO(), address.Scripthash)
	if err != nil {
		logger(c).Error().Err(err).Msg("GetBalance failed")
		return Render(c, http.StatusInternalServerError, templates.PageAddressNotFound())
	}

	history, err := ac.ElectrumClient.GetHistory(context.TODO(), address.Scripthash)
	if err != nil {
		logger(c).Error().Err(err).Msg("ListUnspent failed")
		return Render(c, http.StatusInternalServerError, templates.PageAddressNotFound())
	}

	slices.Reverse(history)

	var txHistory []*electrum.GetTransactionResult

	for _, historyResult := range history {
		if history == nil {
			continue
		}

		tx, err := ac.ElectrumClient.GetTransaction(context.TODO(), historyResult.Hash)
		if err != nil {
			logger(c).Error().Err(err).Msg("GetTransaction failed")
			return Render(c, http.StatusInternalServerError, templates.PageAddressNotFound())
		}

		txHistory = append(txHistory, tx)
	}

	return Render(c, http.StatusOK, templates.PageAddress(*address, unspent, balances, txHistory, history, nil))
}

func (ac AddressController) New(c echo.Context) error {
	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	emails, err := ac.API.GetUserEmails(user.ID)
	if err != nil {
		logger(c).Error().Err(err).Msg("GetUserEmails lookup failed")
	}

	return Render(c, http.StatusOK, templates.PageAddressNew(emails, err))
}

func (ac AddressController) Create(c echo.Context) error {
	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	emails, err := ac.API.GetUserEmails(user.ID)
	if err != nil {
		logger(c).Error().Err(err).Msg("GetUserEmails lookup failed")
	}

	type FormParams struct {
		Address     string `form:"address"`
		Description string `form:"description"`
		Email       string `form:"email"`
		Pubkey      string `form:"pubkey"`
	}

	var params FormParams
	if err := c.Bind(&params); err != nil {
		logger(c).Warn().Err(err).Msg("Unable to parse")
		return Render(c, http.StatusUnprocessableEntity, templates.PageAddressNew(emails, errors.New("Unable to process request.")))
	}

	if len(emails) == 0 {
		if err := validate.Var(params.Email, "required,email"); err != nil {
			logger(c).Warn().Err(err).Msg("email parse error")
			return Render(c, http.StatusUnprocessableEntity, templates.PageAddressNew(emails, errors.New("Please provide an email address. Alternatively, if you don't want to provide an email address, add a webhook (Settings > Webhooks).")))
		}

		if !ac.Config.IsSMTPConfigured() {
			return Render(c, http.StatusUnprocessableEntity, templates.PageAddressNew(emails, errors.New("Unable to save address. SMTP is not configured.")))
		}

		if err := ac.API.CreateEmail(user.ID, params.Email, "", params.Pubkey); err != nil {
			userError := "Unable to save email."

			if err.Error() == "Invalid Pubkey" {
				userError = "Invalid pubkey"
			}
			logger(c).Warn().Err(err).Msg("Unable to create email")
			return Render(c, http.StatusUnprocessableEntity, templates.PageAddressNew(emails, errors.New(userError)))
		}
		email, err := ac.API.GetEmailByAddress(user.ID, params.Email)
		if err != nil {
			logger(c).Warn().Err(err).Msg("Unable get email")
			return Render(c, http.StatusUnprocessableEntity, templates.PageAddressNew(emails, errors.New("Unable to get email")))
		}

		err = ac.EmailClient.SendVerification(email)
		if err != nil {
			logger(c).Warn().Err(err).Msg("Unable to send verification email")
			return Render(c, http.StatusUnprocessableEntity, templates.PageAddressNew(emails, errors.New("Unable to send verification email")))
		}

		emails = []api.Email{email}
	}

	addr := strings.TrimSpace(params.Address)

	if hasValidPrefix(addr) {
		xpub, err := ac.API.CreateXpub(user.ID, addr, &params.Description, ac.Gap)
		if err != nil {
			logger(c).Warn().Err(err).Msg("unable to save xpub")
			return Render(c, http.StatusUnprocessableEntity, templates.PageAddressNew(emails, errors.New("Internal error")))
		}

		addresses := make([]api.Address, ac.Gap*2)
		for i := 0; i < ac.Gap; i++ {
			address, err := lib.GetAddressFromExtPubKey(addr, true, i)
			if err != nil {
				logger(c).Warn().Err(err).Msg("unable to parse pubkey")
				return Render(c, http.StatusUnprocessableEntity, templates.PageAddressNew(emails, errors.New("Internal error")))
			}

			changeAddress, err := lib.GetAddressFromExtPubKey(addr, false, i)
			if err != nil {
				logger(c).Warn().Err(err).Msg("unable to parse change pubkey")
				return Render(c, http.StatusUnprocessableEntity, templates.PageAddressNew(emails, errors.New("Internal error")))
			}

			addresses[i*2] = api.Address{
				Model:        api.Model{ID: uuid.New().String()},
				UserID:       user.ID,
				XpubID:       &xpub.ID,
				Address:      address,
				IsExternal:   true,
				AddressIndex: i,
			}
			addresses[i*2+1] = api.Address{
				Model:        api.Model{ID: uuid.New().String()},
				UserID:       user.ID,
				XpubID:       &xpub.ID,
				Address:      changeAddress,
				IsExternal:   false,
				AddressIndex: i,
			}
		}

		if err := ac.API.CreateAddress(addresses...); err != nil {
			logger(c).Error().Err(err).Msg("unable to save address")
			return Render(c, http.StatusInternalServerError, templates.PageAddressNew(emails, errors.New("Internal error")))
		}

		return c.Redirect(http.StatusSeeOther, "/app")
	}

	addressResp, err := ac.BitcoinClient.GetAddressesFromDescriptor(addr)
	if err != nil {
		if customError, ok := err.(lib.Error); ok {
			return Render(c, http.StatusInternalServerError, templates.PageAddressNew(emails, customError.DisplayError()))
		}
		return Render(c, http.StatusInternalServerError, templates.PageAddressNew(emails, errors.New("Internal error")))
	}

	if len(addressResp) > 0 {
		logger(c).Debug().Msg("Address is a descriptor")
		addresses := make([]api.Address, len(addressResp))
		for i, address := range addressResp {
			addresses[i] = api.Address{
				Model: api.Model{
					ID: uuid.New().String(),
				},
				UserID:  user.ID,
				Address: address,
			}
		}

		if err := ac.API.CreateAddress(addresses...); err != nil {
			logger(c).Error().Err(err).Any("addresses", addresses).Msg("unable to save address")
			return Render(c, http.StatusInternalServerError, templates.PageAddressNew(emails, errors.New("Internal error")))
		}

		return c.Redirect(http.StatusSeeOther, "/app")
	}

	logger(c).Debug().Msg("Address is standard (not descriptor, not xpub)")
	address := api.Address{
		Model:   api.Model{ID: uuid.New().String()},
		UserID:  user.ID,
		Address: addr,
		Name:    nil, // Default to nil
	}
	if params.Description != "" {
		address.Name = &params.Description
	}

	err = ac.API.CreateAddress(address)
	if err != nil {
		logger(c).Error().Err(err).Msg("Unable to save address.")
		return Render(c, http.StatusFound, templates.PageAddressNew(emails, errors.New("Unable to save address.")))
	}

	return c.Redirect(http.StatusSeeOther, "/app")
}

func (ac AddressController) Delete(c echo.Context) error {
	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	err := ac.API.DeleteAddress(user.ID, c.Param("address"))
	if err != nil {
		logger(c).Error().Err(err).Msg("Unable to delete address.")
	}
	return c.Redirect(http.StatusSeeOther, "/app")
}

func (ac AddressController) Status(c echo.Context) error {
	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}
	idsParam := c.QueryParam("ids")
	if idsParam == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ids query param is required"})
	}
	ids := strings.Split(idsParam, ",")
	type resp struct {
		ID         string `json:"id"`
		UtxoCount  int    `json:"utxoCount"`
		BalanceSum uint64 `json:"balance"`
	}
	var out []resp
	for _, id := range ids {
		addr, err := ac.API.GetAddressById(id, user.ID)
		if err != nil || addr == nil || addr.UTXOs == nil {
			continue
		}
		var sum uint64
		for _, utxo := range addr.UTXOs {
			sum += utxo.Value
		}
		out = append(out, resp{
			ID:         id,
			UtxoCount:  len(addr.UTXOs),
			BalanceSum: sum,
		})
	}
	return c.JSON(http.StatusOK, out)
}

func hasValidPrefix(s string) bool {
	return strings.HasPrefix(s, "xpub") ||
		strings.HasPrefix(s, "ypub") ||
		strings.HasPrefix(s, "zpub")
}
