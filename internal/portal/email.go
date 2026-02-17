package portal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/config"
)

const mandrillSendURL = "https://mandrillapp.com/api/1.0/messages/send.json"

func maskEmail(email string) string {
	at := strings.Index(email, "@")
	if at <= 1 {
		return "***"
	}
	return email[:1] + strings.Repeat("*", at-1) + email[at:]
}

type EmailSender interface {
	IsConfigured() bool
	SendProposalNotificationAsync(pluginID, title, contactEmail string)
	SendApprovalNotificationAsync(pluginID, title, contactEmail string)
	SendPublishNotificationAsync(pluginID, title, contactEmail string)
}

type EmailService struct {
	cfg         config.PortalEmailConfig
	portalURL   string
	mandrillURL string
	client      *http.Client
	logger      *logrus.Logger
}

func NewEmailService(cfg config.PortalEmailConfig, portalURL string, logger *logrus.Logger) *EmailService {
	return &EmailService{
		cfg:         cfg,
		portalURL:   strings.TrimRight(portalURL, "/"),
		mandrillURL: mandrillSendURL,
		logger:      logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *EmailService) IsConfigured() bool {
	return s.cfg.IsConfigured()
}

type mandrillMessage struct {
	Key     string              `json:"key"`
	Message mandrillMessageBody `json:"message"`
}

type mandrillMessageBody struct {
	FromEmail string              `json:"from_email"`
	FromName  string              `json:"from_name"`
	To        []mandrillRecipient `json:"to"`
	Subject   string              `json:"subject"`
	HTML      string              `json:"html"`
	Text      string              `json:"text"`
}

type mandrillRecipient struct {
	Email string `json:"email"`
	Type  string `json:"type"`
}

type mandrillSendResult struct {
	Email        string `json:"email"`
	Status       string `json:"status"`
	RejectReason string `json:"reject_reason,omitempty"`
}

func (s *EmailService) SendProposalNotificationAsync(pluginID, title, contactEmail string) {
	if !s.IsConfigured() {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := s.sendProposalNotification(ctx, pluginID, title, contactEmail)
		if err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"plugin_id": pluginID,
			}).Error("failed to send proposal notification email")
		}
	}()
}

func (s *EmailService) sendProposalNotification(ctx context.Context, pluginID, title, contactEmail string) error {
	pid := html.EscapeString(pluginID)
	t := html.EscapeString(title)
	ce := html.EscapeString(contactEmail)

	proposalURL := fmt.Sprintf("%s/admin/plugin-proposals/%s", s.portalURL, url.PathEscape(pluginID))

	subject := fmt.Sprintf("New Plugin Proposal: %s", t)
	htmlBody := fmt.Sprintf(`
<h2>New Plugin Proposal Submitted</h2>
<p>A new plugin proposal has been submitted for review.</p>
<table style="border-collapse: collapse; margin: 20px 0;">
  <tr>
    <td style="padding: 8px; font-weight: bold;">Plugin ID:</td>
    <td style="padding: 8px;">%s</td>
  </tr>
  <tr>
    <td style="padding: 8px; font-weight: bold;">Title:</td>
    <td style="padding: 8px;">%s</td>
  </tr>
  <tr>
    <td style="padding: 8px; font-weight: bold;">Contact Email:</td>
    <td style="padding: 8px;">%s</td>
  </tr>
</table>
<p><a href="%s">View proposal in admin portal</a></p>
`, pid, t, ce, html.EscapeString(proposalURL))

	text := fmt.Sprintf(`New Plugin Proposal Submitted

Plugin ID: %s
Title: %s
Contact Email: %s

View proposal: %s
`, pluginID, title, contactEmail, proposalURL)

	return s.sendToAdmins(ctx, subject, htmlBody, text)
}

// TODO: migrate async methods to use Redis/Asynq queue for reliability and retries
func (s *EmailService) SendApprovalNotificationAsync(pluginID, title, contactEmail string) {
	if !s.IsConfigured() || contactEmail == "" {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := s.sendApprovalNotification(ctx, pluginID, title, contactEmail)
		if err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"plugin_id": pluginID,
				"email":     maskEmail(contactEmail),
			}).Error("failed to send approval notification email")
		}
	}()
}

func (s *EmailService) sendApprovalNotification(ctx context.Context, pluginID, title, contactEmail string) error {
	t := html.EscapeString(title)

	subject := fmt.Sprintf("Your Plugin Proposal Has Been Approved: %s", t)
	htmlBody := fmt.Sprintf(`
<h2>Plugin Proposal Approved</h2>
<p>Your plugin proposal <strong>%s</strong> has been approved.</p>
<p>To complete the listing process:</p>
<ol>
  <li>Pay the listing fee through the developer portal</li>
  <li>Once payment is confirmed, your plugin will be published automatically</li>
</ol>
<p>Thank you for contributing to Vultisig!</p>
`, t)

	text := fmt.Sprintf(`Plugin Proposal Approved

Your plugin proposal "%s" has been approved.

To complete the listing process:
1. Pay the listing fee through the developer portal
2. Once payment is confirmed, your plugin will be published automatically

Thank you for contributing to Vultisig!
`, title)

	return s.sendToRecipient(ctx, contactEmail, subject, htmlBody, text)
}

func (s *EmailService) SendPublishNotificationAsync(pluginID, title, contactEmail string) {
	if !s.IsConfigured() || contactEmail == "" {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := s.sendPublishNotification(ctx, pluginID, title, contactEmail)
		if err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"plugin_id": pluginID,
				"email":     maskEmail(contactEmail),
			}).Error("failed to send publish notification email")
		}
	}()
}

func (s *EmailService) sendPublishNotification(ctx context.Context, pluginID, title, contactEmail string) error {
	t := html.EscapeString(title)
	pid := html.EscapeString(pluginID)
	pluginURL := fmt.Sprintf("%s/plugins/%s", s.portalURL, url.PathEscape(pluginID))

	subject := fmt.Sprintf("Your Plugin Is Now Live: %s", t)
	htmlBody := fmt.Sprintf(`
<h2>Plugin Published!</h2>
<p>Your plugin <strong>%s</strong> is now live on the Vultisig marketplace.</p>
<p><a href="%s">View your plugin</a></p>
<p>Plugin ID: %s</p>
<p>Thank you for contributing to Vultisig!</p>
`, t, html.EscapeString(pluginURL), pid)

	text := fmt.Sprintf(`Plugin Published!

Your plugin "%s" is now live on the Vultisig marketplace.

View your plugin: %s

Plugin ID: %s

Thank you for contributing to Vultisig!
`, title, pluginURL, pluginID)

	return s.sendToRecipient(ctx, contactEmail, subject, htmlBody, text)
}

func (s *EmailService) sendToAdmins(ctx context.Context, subject, htmlBody, text string) error {
	recipients := make([]mandrillRecipient, len(s.cfg.NotificationEmails))
	for i, email := range s.cfg.NotificationEmails {
		recipients[i] = mandrillRecipient{Email: email, Type: "to"}
	}
	return s.sendEmail(ctx, recipients, subject, htmlBody, text)
}

func (s *EmailService) sendToRecipient(ctx context.Context, email, subject, htmlBody, text string) error {
	recipients := []mandrillRecipient{{Email: email, Type: "to"}}
	return s.sendEmail(ctx, recipients, subject, htmlBody, text)
}

func (s *EmailService) sendEmail(ctx context.Context, recipients []mandrillRecipient, subject, htmlBody, text string) error {
	msg := mandrillMessage{
		Key: s.cfg.MandrillAPIKey,
		Message: mandrillMessageBody{
			FromEmail: s.cfg.FromEmail,
			FromName:  s.cfg.FromName,
			To:        recipients,
			Subject:   subject,
			HTML:      htmlBody,
			Text:      text,
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal email request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.mandrillURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create email request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send email request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mandrill returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var results []mandrillSendResult
	err = json.Unmarshal(respBody, &results)
	if err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	for _, r := range results {
		if r.Status != "sent" && r.Status != "queued" {
			return fmt.Errorf("email to %s failed: %s (%s)", maskEmail(r.Email), r.Status, r.RejectReason)
		}
	}

	return nil
}
