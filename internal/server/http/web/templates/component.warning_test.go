package templates

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/jpcummins/satwatch/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestCheckConfigurationWarning(t *testing.T) {
	tests := []struct {
		name           string
		webhooks       []api.Webhook
		emails         []api.Email
		wantNoNotif    bool
		wantUnverified bool
	}{
		{
			name:           "no webhooks or emails configured",
			webhooks:       []api.Webhook{},
			emails:         []api.Email{},
			wantNoNotif:    true,
			wantUnverified: false,
		},
		{
			name: "1 webhook defined, 0 emails",
			webhooks: []api.Webhook{
				{Model: api.Model{ID: "1"}},
			},
			emails:         []api.Email{},
			wantNoNotif:    false,
			wantUnverified: false,
		},
		{
			name:     "1 unverified email, 0 webhooks",
			webhooks: []api.Webhook{},
			emails: []api.Email{
				{Model: api.Model{ID: "1"}, Email: "test@example.com", IsVerified: false},
			},
			wantNoNotif:    false,
			wantUnverified: true,
		},
		{
			name:     "1 verified email, 0 webhooks",
			webhooks: []api.Webhook{},
			emails: []api.Email{
				{Model: api.Model{ID: "1"}, Email: "test@example.com", IsVerified: true},
			},
			wantNoNotif:    false,
			wantUnverified: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Render the template
			var buf bytes.Buffer
			err := CheckConfigurationWarning(tt.webhooks, tt.emails).Render(context.Background(), &buf)
			assert.NoError(t, err, "Template should render without error")
			html := buf.String()

			// Check for no notifications warning
			if tt.wantNoNotif {
				assert.Contains(t, html, "data-testid=\"no-notifications-warning\"", "Should show no notifications warning")
			} else {
				assert.NotContains(t, html, "data-testid=\"no-notifications-warning\"", "Should not show no notifications warning")
			}

			// Check for unverified email warning
			if tt.wantUnverified {
				assert.Contains(t, html, "data-testid=\"unverified-email-warning\"", "Should show unverified email warning")
				assert.Contains(t, html, tt.emails[0].Email, "Should include the email address")
				assert.Contains(t, html, fmt.Sprintf("href=\"/app/settings/email/%s/verify\"", tt.emails[0].ID), "Should have correct verification link")
			} else {
				assert.NotContains(t, html, "data-testid=\"unverified-email-warning\"", "Should not show unverified email warning")
			}
		})
	}
}
