package clients

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/jpcummins/satwatch/internal/api"
	"github.com/jpcummins/satwatch/internal/configs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wneessen/go-mail"
)

type DbLogger interface {
	LogAlertEmail(email string, user_id string, address_id string, transaction_id string, err error) error
	GetDailyEmailCount(address_id string, user_id string) (int, error)
}

const (
	MAX_EMAIL_PER_ADDRESS_PER_DAY_WARNING = 7
	MAX_EMAIL_PER_ADDRESS_PER_DAY         = 10
)

type MailClient struct {
	log      zerolog.Logger
	dbLogger DbLogger
	url      string
	config   *configs.Config
}

func NewMailClient(dbLogger DbLogger, url string, config *configs.Config) (MailClient, error) {
	return MailClient{
		log:      log.With().Str("module", "mailer").Logger(),
		dbLogger: dbLogger,
		url:      url,
		config:   config,
	}, nil
}

type NotificationData struct {
	Address   api.Address
	Tx        string
	Amount    int
	Sent      bool
	Confirmed bool
}

func (m MailClient) SendVerification(email api.Email) error {
	m.log.Debug().Str("id", email.ID).Msg("SendVerification")

	t := "internal/clients/templates/email.verify.tmpl"
	data, _ := os.ReadFile(t)

	tmpl, err := template.New(t).Parse(string(data))
	if err != nil {
		m.log.Error().Str("path", t).Str("id", email.ID).Err(err).Msg("Failed to parse email template")
	}

	type templateData struct {
		Email api.Email
		Host  string
		Year  string
	}

	td := templateData{
		Email: email,
		Host:  m.url,
		Year:  strconv.Itoa(time.Now().Year()),
	}

	m.log.Debug().Any("data", td).Msg("Rendering verification template")

	var result bytes.Buffer
	err = tmpl.Execute(&result, td)
	if err != nil {
		m.log.Error().Str("path", t).Str("id", email.ID).Err(err).Msg("Failed to render email template")
	}

	return m.sendEmail(m.config.SmtpFrom, email, "Verify your email for sat.watch", result.String())
}

func (m MailClient) SendNotification(email api.Email, data NotificationData, address api.Address) {
	count, err := m.dbLogger.GetDailyEmailCount(address.ID, address.UserID)
	if err != nil {
		m.log.Error().Err(err).Msg("Failed to get daily email count")
	}

	m.log.Info().Int("count", count).Msg("emails")

	showRateLimt := false
	if count > MAX_EMAIL_PER_ADDRESS_PER_DAY_WARNING {
		showRateLimt = true
	}

	if count > MAX_EMAIL_PER_ADDRESS_PER_DAY {
		m.log.Info().Str("user_id", address.UserID).Str("address_id", address.ID).Str("tx", data.Tx).Msg("Reached daily alert limit")
		_ = m.dbLogger.LogAlertEmail(email.Email, address.UserID, address.ID, data.Tx, errors.New("rate limit"))
		return
	}

	t := "internal/clients/templates/email.transaction-alert.tmpl"
	fileData, _ := os.ReadFile(t)

	tmpl, err := template.New(t).Parse(string(fileData))
	if err != nil {
		m.log.Error().Str("path", t).Str("template", t).Err(err).Msg("Failed to parse email template")
	}

	type ExtendedNotificationData struct {
		NotificationData
		Year                 string
		Host                 string
		ShowRateLimitWarning bool
		RemainingAlerts      int
	}

	extData := ExtendedNotificationData{
		NotificationData:     data,
		Year:                 strconv.Itoa(time.Now().Year()),
		Host:                 m.url,
		ShowRateLimitWarning: showRateLimt,
		RemainingAlerts:      MAX_EMAIL_PER_ADDRESS_PER_DAY - count,
	}

	var result bytes.Buffer
	err = tmpl.Execute(&result, extData)
	if err != nil {
		m.log.Error().Str("path", t).Err(err).Msg("Failed to render email template")
	}

	err = m.sendEmail(m.config.SmtpFrom, email, "New Transaction Alert", result.String())
	if err != nil {
		m.log.Err(err).Msg("Unable to send mail")
	}

	err = m.dbLogger.LogAlertEmail(email.Email, address.UserID, address.ID, data.Tx, err)
	if err != nil {
		m.log.Err(err).Msg("Unable to log alert email")
	}
}

func (m MailClient) sendEmail(from string, to api.Email, subject string, htmlBody string) error {
	log.Debug().Any("email", to).Msg("sending")
	if to.Pubkey == nil || *to.Pubkey == "" {
		return m.sendEmailUnencrypted(from, to, subject, htmlBody)
	}

	return m.sendEmailEncrypted(from, to, subject, htmlBody)
}

func (m MailClient) sendEmailUnencrypted(from string, to api.Email, subject string, htmlBody string) error {
	msg := mail.NewMsg()

	if err := msg.From(from); err != nil {
		return err
	}

	if err := msg.To(to.Email); err != nil {
		return err
	}

	msg.SetUserAgent("sat.watch mailer")
	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextHTML, htmlBody)

	host, port, user, password := m.config.GetSMTPConfig()
	c, err := mail.NewClient(host, mail.WithPort(port), mail.WithSMTPAuth(mail.SMTPAuthPlain), mail.WithUsername(user), mail.WithPassword(password))
	if err != nil {
		return err
	}

	m.log.Debug().Str("body", htmlBody).Msg("Sending email")
	if err := c.DialAndSend(msg); err != nil {
		m.log.Error().Err(err).Msg("Unable to send message")
	} else {
		m.log.Debug().Msg("Message sent")
	}

	return nil
}

func (m MailClient) sendEmailEncrypted(from string, to api.Email, subject string, htmlBody string) error {
	subMessage := buildHTMLSubmessage(htmlBody)

	encryptedBlock, err := encryptWithPGP(*to.Pubkey, subMessage)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to encrypt message: %v", err))
	}

	mimeBody, boundary, err := buildMultipartEncrypted(encryptedBlock)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to build MIME structure: %v", err))
	}

	msg := buildEmail(from, to.Email, subject, boundary, mimeBody)

	host, port, user, password := m.config.GetSMTPConfig()
	serverAddr := fmt.Sprintf("%s:%d", host, port)

	auth := smtp.PlainAuth("", user, password, host)

	tlsConfig := &tls.Config{
		ServerName: host,
	}

	c, err := smtp.Dial(serverAddr)
	if err != nil {
		return errors.New(fmt.Sprintf("Dial error: %v", err))
	}
	defer c.Close()

	// If the server supports the STARTTLS extension:
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err = c.StartTLS(tlsConfig); err != nil {
			return errors.New(fmt.Sprintf("StartTLS error: %v", err))
		}
	}

	// Now authenticate
	if ok, _ := c.Extension("AUTH"); ok {
		if err = c.Auth(auth); err != nil {
			return errors.New(fmt.Sprintf("AUTH error: %v", err))
		}
	}

	// Set the envelope addresses
	if err = c.Mail(from); err != nil {
		return errors.New(fmt.Sprintf("MAIL FROM error: %v", err))
	}
	if err = c.Rcpt(to.Email); err != nil {
		return errors.New(fmt.Sprintf("RCPT TO error: %v", err))
	}

	// Data command
	wc, err := c.Data()
	if err != nil {
		return errors.New(fmt.Sprintf("DATA error: %v", err))
	}

	// Write the entire message
	_, err = wc.Write(msg)
	if err != nil {
		return errors.New(fmt.Sprintf("Write to SMTP server error: %v", err))
	}
	err = wc.Close()
	if err != nil {
		return errors.New(fmt.Sprintf("Close error: %v", err))
	}

	// Finally, quit
	if err = c.Quit(); err != nil {
		return errors.New(fmt.Sprintf("QUIT error: %v", err))
	}

	m.log.Info().Msg("Email sent")
	return nil
}

func buildHTMLSubmessage(htmlContent string) []byte {
	// Minimal example (UTF-8, 8bit). For robust usage, consider quoted-printable or base64.
	return []byte(fmt.Sprintf(
		`Content-Type: text/html; charset="UTF-8"
Content-Transfer-Encoding: 8bit

%s
`, htmlContent))
}

func encryptWithPGP(pubKey string, subMessage []byte) (string, error) {
	pgp := crypto.PGP()
	publicKey, err := crypto.NewKeyFromArmored(pubKey)
	encHandle, err := pgp.Encryption().Recipient(publicKey).New()
	pgpMessage, err := encHandle.Encrypt(subMessage)
	armored, err := pgpMessage.ArmorBytes()
	return string(armored), err
}

func buildMultipartEncrypted(armoredCiphertext string) ([]byte, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Part 1: application/pgp-encrypted
	partAHeader := textproto.MIMEHeader{}
	partAHeader.Set("Content-Type", "application/pgp-encrypted")
	partA, err := writer.CreatePart(partAHeader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create part A: %w", err)
	}
	// Typically just "Version: 1"
	_, err = partA.Write([]byte("Version: 1\n"))
	if err != nil {
		return nil, "", err
	}

	// Part 2: application/octet-stream (the actual ASCII-armored ciphertext)
	partBHeader := textproto.MIMEHeader{}
	partBHeader.Set("Content-Type", "application/octet-stream")
	partB, err := writer.CreatePart(partBHeader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create part B: %w", err)
	}
	_, err = partB.Write([]byte(armoredCiphertext))
	if err != nil {
		return nil, "", err
	}

	// Close the writer to finalize
	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return buf.Bytes(), writer.Boundary(), nil
}

func buildEmail(from, to, subject, boundary string, mimeBody []byte) []byte {
	// Build standard email headers
	headers := []string{
		"MIME-Version: 1.0",
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		fmt.Sprintf("Date: %s", time.Now().Format(time.RFC1123Z)),
		fmt.Sprintf(`Content-Type: multipart/encrypted; boundary="%s"; protocol="application/pgp-encrypted"`, boundary),
	}

	// Combine headers + a blank line + the MIME body
	var msg bytes.Buffer
	msg.WriteString(strings.Join(headers, "\r\n"))
	msg.WriteString("\r\n\r\n") // blank line after headers
	msg.Write(mimeBody)

	return msg.Bytes()
}
