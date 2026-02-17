package portal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/config"
)

type MockEmailSender struct {
	mu                         sync.Mutex
	ProposalNotifications      []EmailNotification
	ApprovalNotifications      []EmailNotification
	PublishNotifications       []EmailNotification
	configured                 bool
	SendProposalNotificationFn func(pluginID, title, contactEmail string)
	SendApprovalNotificationFn func(pluginID, title, contactEmail string)
	SendPublishNotificationFn  func(pluginID, title, contactEmail string)
}

type EmailNotification struct {
	PluginID     string
	Title        string
	ContactEmail string
}

func NewMockEmailSender(configured bool) *MockEmailSender {
	return &MockEmailSender{
		configured: configured,
	}
}

func (m *MockEmailSender) IsConfigured() bool {
	return m.configured
}

func (m *MockEmailSender) SendProposalNotificationAsync(pluginID, title, contactEmail string) {
	if m.SendProposalNotificationFn != nil {
		m.SendProposalNotificationFn(pluginID, title, contactEmail)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ProposalNotifications = append(m.ProposalNotifications, EmailNotification{
		PluginID:     pluginID,
		Title:        title,
		ContactEmail: contactEmail,
	})
}

func (m *MockEmailSender) SendApprovalNotificationAsync(pluginID, title, contactEmail string) {
	if m.SendApprovalNotificationFn != nil {
		m.SendApprovalNotificationFn(pluginID, title, contactEmail)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ApprovalNotifications = append(m.ApprovalNotifications, EmailNotification{
		PluginID:     pluginID,
		Title:        title,
		ContactEmail: contactEmail,
	})
}

func (m *MockEmailSender) SendPublishNotificationAsync(pluginID, title, contactEmail string) {
	if m.SendPublishNotificationFn != nil {
		m.SendPublishNotificationFn(pluginID, title, contactEmail)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PublishNotifications = append(m.PublishNotifications, EmailNotification{
		PluginID:     pluginID,
		Title:        title,
		ContactEmail: contactEmail,
	})
}

func (m *MockEmailSender) GetProposalNotifications() []EmailNotification {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]EmailNotification{}, m.ProposalNotifications...)
}

func (m *MockEmailSender) GetApprovalNotifications() []EmailNotification {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]EmailNotification{}, m.ApprovalNotifications...)
}

func (m *MockEmailSender) GetPublishNotifications() []EmailNotification {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]EmailNotification{}, m.PublishNotifications...)
}

func TestMockEmailSender_Interface(t *testing.T) {
	var _ EmailSender = (*MockEmailSender)(nil)
	var _ EmailSender = (*EmailService)(nil)
}

func TestMockEmailSender_SendProposalNotification(t *testing.T) {
	mock := NewMockEmailSender(true)

	mock.SendProposalNotificationAsync("test-plugin-001", "Test Plugin", "dev@example.com")

	notifications := mock.GetProposalNotifications()
	if len(notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifications))
	}

	n := notifications[0]
	if n.PluginID != "test-plugin-001" {
		t.Errorf("expected pluginID 'test-plugin-001', got '%s'", n.PluginID)
	}
	if n.Title != "Test Plugin" {
		t.Errorf("expected title 'Test Plugin', got '%s'", n.Title)
	}
	if n.ContactEmail != "dev@example.com" {
		t.Errorf("expected contactEmail 'dev@example.com', got '%s'", n.ContactEmail)
	}
}

func TestMockEmailSender_SendApprovalNotification(t *testing.T) {
	mock := NewMockEmailSender(true)

	mock.SendApprovalNotificationAsync("test-plugin-002", "Approved Plugin", "approved@example.com")

	notifications := mock.GetApprovalNotifications()
	if len(notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifications))
	}

	n := notifications[0]
	if n.PluginID != "test-plugin-002" {
		t.Errorf("expected pluginID 'test-plugin-002', got '%s'", n.PluginID)
	}
	if n.Title != "Approved Plugin" {
		t.Errorf("expected title 'Approved Plugin', got '%s'", n.Title)
	}
	if n.ContactEmail != "approved@example.com" {
		t.Errorf("expected contactEmail 'approved@example.com', got '%s'", n.ContactEmail)
	}
}

func TestMockEmailSender_SendPublishNotification(t *testing.T) {
	mock := NewMockEmailSender(true)

	mock.SendPublishNotificationAsync("test-plugin-003", "Published Plugin", "published@example.com")

	notifications := mock.GetPublishNotifications()
	if len(notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifications))
	}

	n := notifications[0]
	if n.PluginID != "test-plugin-003" {
		t.Errorf("expected pluginID 'test-plugin-003', got '%s'", n.PluginID)
	}
	if n.Title != "Published Plugin" {
		t.Errorf("expected title 'Published Plugin', got '%s'", n.Title)
	}
	if n.ContactEmail != "published@example.com" {
		t.Errorf("expected contactEmail 'published@example.com', got '%s'", n.ContactEmail)
	}
}

func TestEmailService_IsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.PortalEmailConfig
		expected bool
	}{
		{
			name:     "not configured - empty config",
			cfg:      config.PortalEmailConfig{},
			expected: false,
		},
		{
			name: "not configured - missing api key",
			cfg: config.PortalEmailConfig{
				FromEmail:          "noreply@vultisig.com",
				NotificationEmails: []string{"admin@vultisig.com"},
			},
			expected: false,
		},
		{
			name: "not configured - missing from email",
			cfg: config.PortalEmailConfig{
				MandrillAPIKey:     "test-api-key",
				NotificationEmails: []string{"admin@vultisig.com"},
			},
			expected: false,
		},
		{
			name: "not configured - missing notification emails",
			cfg: config.PortalEmailConfig{
				MandrillAPIKey: "test-api-key",
				FromEmail:      "noreply@vultisig.com",
			},
			expected: false,
		},
		{
			name: "configured",
			cfg: config.PortalEmailConfig{
				MandrillAPIKey:     "test-api-key",
				FromEmail:          "noreply@vultisig.com",
				FromName:           "Vultisig",
				NotificationEmails: []string{"admin@vultisig.com"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewEmailService(tt.cfg, "https://portal.vultisig.com", logrus.New())
			if svc.IsConfigured() != tt.expected {
				t.Errorf("IsConfigured() = %v, expected %v", svc.IsConfigured(), tt.expected)
			}
		})
	}
}

func TestEmailService_SendProposalNotification_NotConfigured(t *testing.T) {
	svc := NewEmailService(config.PortalEmailConfig{}, "https://portal.vultisig.com", logrus.New())

	svc.SendProposalNotificationAsync("test-plugin", "Test", "test@example.com")
}

func TestEmailService_SendApprovalNotification_EmptyEmail(t *testing.T) {
	cfg := config.PortalEmailConfig{
		MandrillAPIKey:     "test-api-key",
		FromEmail:          "noreply@vultisig.com",
		NotificationEmails: []string{"admin@vultisig.com"},
	}
	svc := NewEmailService(cfg, "https://portal.vultisig.com", logrus.New())

	svc.SendApprovalNotificationAsync("test-plugin", "Test", "")
}

func TestEmailService_SendPublishNotification_EmptyEmail(t *testing.T) {
	cfg := config.PortalEmailConfig{
		MandrillAPIKey:     "test-api-key",
		FromEmail:          "noreply@vultisig.com",
		NotificationEmails: []string{"admin@vultisig.com"},
	}
	svc := NewEmailService(cfg, "https://portal.vultisig.com", logrus.New())

	svc.SendPublishNotificationAsync("test-plugin", "Test", "")
}

func TestEmailService_SendEmail_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var msg mandrillMessage
		err := json.NewDecoder(r.Body).Decode(&msg)
		if err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		if msg.Key != "test-api-key" {
			t.Errorf("expected API key 'test-api-key', got '%s'", msg.Key)
		}
		if msg.Message.FromEmail != "noreply@vultisig.com" {
			t.Errorf("expected from email 'noreply@vultisig.com', got '%s'", msg.Message.FromEmail)
		}
		if len(msg.Message.To) != 1 {
			t.Errorf("expected 1 recipient, got %d", len(msg.Message.To))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]mandrillSendResult{
			{Email: msg.Message.To[0].Email, Status: "sent"},
		})
	}))
	defer server.Close()

	cfg := config.PortalEmailConfig{
		MandrillAPIKey:     "test-api-key",
		FromEmail:          "noreply@vultisig.com",
		FromName:           "Vultisig",
		NotificationEmails: []string{"admin@vultisig.com"},
	}
	svc := NewEmailService(cfg, "https://portal.vultisig.com", logrus.New())
	svc.client = server.Client()

	originalURL := mandrillSendURL
	defer func() {
		if mandrillSendURL != originalURL {
			t.Error("mandrillSendURL should not be changed")
		}
	}()

	ctx := context.Background()
	recipients := []mandrillRecipient{{Email: "test@example.com", Type: "to"}}

	err := svc.sendEmailTo(ctx, server.URL, recipients, "Test Subject", "<p>HTML</p>", "Text")
	if err != nil {
		t.Errorf("sendEmailTo failed: %v", err)
	}
}

func TestEmailService_SendEmail_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status":"error","message":"Invalid API key"}`))
	}))
	defer server.Close()

	cfg := config.PortalEmailConfig{
		MandrillAPIKey:     "invalid-key",
		FromEmail:          "noreply@vultisig.com",
		NotificationEmails: []string{"admin@vultisig.com"},
	}
	svc := NewEmailService(cfg, "https://portal.vultisig.com", logrus.New())
	svc.client = server.Client()

	ctx := context.Background()
	recipients := []mandrillRecipient{{Email: "test@example.com", Type: "to"}}

	err := svc.sendEmailTo(ctx, server.URL, recipients, "Test", "<p>HTML</p>", "Text")
	if err == nil {
		t.Error("expected error for invalid API key")
	}
}

func TestEmailService_SendEmail_RejectedEmail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]mandrillSendResult{
			{Email: "invalid@example.com", Status: "rejected", RejectReason: "invalid-sender"},
		})
	}))
	defer server.Close()

	cfg := config.PortalEmailConfig{
		MandrillAPIKey:     "test-api-key",
		FromEmail:          "noreply@vultisig.com",
		NotificationEmails: []string{"admin@vultisig.com"},
	}
	svc := NewEmailService(cfg, "https://portal.vultisig.com", logrus.New())
	svc.client = server.Client()

	ctx := context.Background()
	recipients := []mandrillRecipient{{Email: "invalid@example.com", Type: "to"}}

	err := svc.sendEmailTo(ctx, server.URL, recipients, "Test", "<p>HTML</p>", "Text")
	if err == nil {
		t.Error("expected error for rejected email")
	}
}

func TestEmailService_SendProposalNotification_Async(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()

		var msg mandrillMessage
		json.NewDecoder(r.Body).Decode(&msg)

		if msg.Message.Subject == "" {
			t.Error("expected non-empty subject")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]mandrillSendResult{
			{Email: msg.Message.To[0].Email, Status: "queued"},
		})
	}))
	defer server.Close()

	cfg := config.PortalEmailConfig{
		MandrillAPIKey:     "test-api-key",
		FromEmail:          "noreply@vultisig.com",
		FromName:           "Vultisig",
		NotificationEmails: []string{"admin@vultisig.com"},
	}
	svc := NewEmailService(cfg, "https://portal.vultisig.com", logrus.New())
	svc.client = server.Client()
	svc.mandrillURL = server.URL

	svc.SendProposalNotificationAsync("test-plugin", "Test Plugin", "dev@example.com")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("timeout waiting for async email send")
	}
}

func (s *EmailService) sendEmailTo(ctx context.Context, url string, recipients []mandrillRecipient, subject, htmlBody, text string) error {
	s.mandrillURL = url
	return s.sendEmail(ctx, recipients, subject, htmlBody, text)
}
