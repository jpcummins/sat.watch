package actions

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/jpcummins/satwatch/internal/api"
	"github.com/jpcummins/satwatch/internal/monitor"
)

type notifier struct {
	API    *api.API
	logger zerolog.Logger
}

func InitWebhookNotifier(appAPI *api.API, appMonitor *monitor.TxMonitor) {
	logger := log.With().Str("module", "webhook").Logger()
	logger.Info().Msg("initializing webhook notifier")
	go func(api *api.API, monitor *monitor.TxMonitor, logger zerolog.Logger) {
		notifier := notifier{API: api, logger: logger}
		notificationStream := monitor.Subscribe()

		for {
			select {
			case tx := <-notificationStream:
				notifier.notify(tx)
			}
		}
	}(appAPI, appMonitor, logger)
	logger.Info().Msg("finished initializing webhook notifier")
}

func (n notifier) notify(notification monitor.TxNotification) error {
	n.logger.Debug().Msg("notified")
	json, err := json.Marshal(notification)
	if err != nil {
		n.logger.Error().Err(err).Msg("unable to parse notificaiton")
		return err
	}

	for _, address := range notification.MatchedAddress {
		n.logger.Debug().Str("address", address.ID).Str("user", address.UserID).Msg("Sending webhook")

		webhooks, err := n.API.GetUserWebhooks(address.UserID)
		if err != nil {
			n.logger.Error().Err(err).Msg("Unable to query user webhooks")
			continue
		}

		if len(webhooks) == 0 {
			n.logger.Info().Msg("No webhooks found for user")
			continue
		}

		for _, webhook := range webhooks {
			body := io.NopCloser(bytes.NewBuffer(json))
			_, err := http.Post(webhook.Url, "application/json", body)
			if err != nil {
				n.logger.Error().Err(err).Any("webhook", webhook).Msg("error calling webhook")
			} else {
				n.logger.Debug().Any("webhook", webhook).Msg("called")
			}
		}
	}

	return nil
}
