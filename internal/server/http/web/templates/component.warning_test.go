package templates

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/jpcummins/satwatch/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestNoNotificationsWarning(t *testing.T) {
	tests := []struct {
		name     string
		webhooks []api.Webhook
		emails   []api.Email
		wantShow bool
	}{
		{
			name:     "no webhooks or emails configured",
			webhooks: []api.Webhook{},
			emails:   []api.Email{},
			wantShow: true,
		},
		{
			name: "1 webhook defined, 0 emails",
			webhooks: []api.Webhook{
				{Model: api.Model{ID: "1"}},
			},
			emails:   []api.Email{},
			wantShow: false,
		},
		{
			name:     "1 unverified email, 0 webhooks",
			webhooks: []api.Webhook{},
			emails: []api.Email{
				{Model: api.Model{ID: "1"}, Email: "test@example.com", IsVerified: false},
			},
			wantShow: false,
		},
		{
			name:     "1 verified email, 0 webhooks",
			webhooks: []api.Webhook{},
			emails: []api.Email{
				{Model: api.Model{ID: "1"}, Email: "test@example.com", IsVerified: true},
			},
			wantShow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The warning should show if there are no webhooks and no emails
			shouldShow := len(tt.webhooks) == 0 && len(tt.emails) == 0
			assert.Equal(t, tt.wantShow, shouldShow, "NoNotificationsWarning visibility mismatch")

			// Test the actual template rendering
			if shouldShow {
				// Render the template
				var buf bytes.Buffer
				err := NoNotificationsWarning().Render(context.Background(), &buf)
				assert.NoError(t, err, "Template should render without error")
				html := buf.String()

				// Verify the rendered HTML contains expected elements
				assert.Contains(t, html, "data-testid=\"no-notifications-warning\"", "Warning should have test ID")
				assert.Contains(t, html, "role=\"alert\"", "Warning should have correct ARIA role")
				assert.Contains(t, html, "Please add an email address or webhook to recieve notifications", "Warning should have correct message")
				assert.Contains(t, html, "href=\"/app/settings\"", "Warning should have correct settings link")
				assert.Contains(t, html, "data-dismiss-target=\"#alert-smtp\"", "Warning should have correct dismiss target")
			}
		})
	}
}

func TestUnverifiedEmailWarning(t *testing.T) {
	tests := []struct {
		name     string
		webhooks []api.Webhook
		emails   []api.Email
		wantShow bool
	}{
		{
			name:     "no webhooks or emails configured",
			webhooks: []api.Webhook{},
			emails:   []api.Email{},
			wantShow: false,
		},
		{
			name: "1 webhook defined, 0 emails",
			webhooks: []api.Webhook{
				{Model: api.Model{ID: "1"}},
			},
			emails:   []api.Email{},
			wantShow: false,
		},
		{
			name:     "1 unverified email, 0 webhooks",
			webhooks: []api.Webhook{},
			emails: []api.Email{
				{Model: api.Model{ID: "1"}, Email: "test@example.com", IsVerified: false},
			},
			wantShow: true,
		},
		{
			name:     "1 verified email, 0 webhooks",
			webhooks: []api.Webhook{},
			emails: []api.Email{
				{Model: api.Model{ID: "1"}, Email: "test@example.com", IsVerified: true},
			},
			wantShow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The warning should show if there are emails but none are verified
			hasVerifiedEmail := false
			for _, email := range tt.emails {
				if email.IsVerified {
					hasVerifiedEmail = true
					break
				}
			}
			shouldShow := len(tt.emails) > 0 && !hasVerifiedEmail
			assert.Equal(t, tt.wantShow, shouldShow, "UnverifiedEmailWarning visibility mismatch")

			// Test the actual template rendering
			if shouldShow {
				// Render the template
				var buf bytes.Buffer
				err := UnverifiedEmailWarning(tt.emails).Render(context.Background(), &buf)
				assert.NoError(t, err, "Template should render without error")
				html := buf.String()

				// Verify the rendered HTML contains expected elements
				assert.Contains(t, html, "data-testid=\"unverified-email-warning\"", "Warning should have test ID")
				assert.Contains(t, html, "role=\"alert\"", "Warning should have correct ARIA role")
				assert.Contains(t, html, "Please check your email", "Warning should have correct message")
				assert.Contains(t, html, tt.emails[0].Email, "Warning should include the email address")
				assert.Contains(t, html, fmt.Sprintf("href=\"/app/settings/email/%s/verify\"", tt.emails[0].ID), "Warning should have correct verification link")
				assert.Contains(t, html, "data-dismiss-target=\"#alert-4\"", "Warning should have correct dismiss target")
			}
		})
	}
}
