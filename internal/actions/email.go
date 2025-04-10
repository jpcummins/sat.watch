package actions

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/jpcummins/satwatch/internal/api"
	"github.com/jpcummins/satwatch/internal/clients"
	"github.com/jpcummins/satwatch/internal/monitor"
)

type emailNotifier struct {
	API         *api.API
	emailClient MailClient
	logger      zerolog.Logger
}

type MailClient interface {
	SendNotification(email api.Email, data clients.NotificationData, address api.Address)
}

func InitEmailNotifier(appAPI *api.API, appMonitor *monitor.TxMonitor, emailClient MailClient) {
	logger := log.With().Str("module", "email").Logger()
	logger.Info().Msg("initializing email notifier")
	go func(api *api.API, monitor *monitor.TxMonitor, emailClient MailClient, logger zerolog.Logger) {
		notifier := emailNotifier{API: api, logger: logger, emailClient: emailClient}
		notificationStream := monitor.Subscribe()

		for {
			select {
			case tx := <-notificationStream:
				notifier.notify(tx)
			}
		}
	}(appAPI, appMonitor, emailClient, logger)
	logger.Info().Msg("finished initializing email notifier")
}

func (n emailNotifier) notify(notification monitor.TxNotification) error {
	n.logger.Debug().Msg("notified")
	for _, address := range notification.MatchedAddress {
		n.logger.Debug().Str("address", address.ID).Str("user", address.UserID).Msg("Sending email")

		emails, err := n.API.GetVerifiedUserEmails(address.UserID)
		if err != nil {
			n.logger.Error().Err(err).Msg("Unable to query user emails")
			continue
		}

		if len(emails) == 0 {
			n.logger.Info().Msg("No verified emails found for user")
			continue
		}

		data := clients.NotificationData{
			Address:   address,
			Tx:        notification.Tx.TxID(),
			Amount:    notification.Amount,
			Sent:      notification.Sent,
			Confirmed: notification.Confirmed,
		}

		for _, email := range emails {
			n.logger.Debug().Str("email", email.Email).Msg("Sending")
			n.emailClient.SendNotification(email, data, address)
		}
	}
	return nil
}
