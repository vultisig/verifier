package portal

import (
	"sync"
	"testing"

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
	logger := logrus.New()

	t.Run("not configured - nil queue client", func(t *testing.T) {
		cfg := config.PortalEmailConfig{
			NotificationEmails: []string{"admin@vultisig.com"},
		}
		svc := NewEmailService(cfg, "https://portal.vultisig.com", nil, logger)
		if svc.IsConfigured() {
			t.Error("expected not configured with nil queue client")
		}
	})

	t.Run("not configured - missing notification emails", func(t *testing.T) {
		cfg := config.PortalEmailConfig{}
		svc := NewEmailService(cfg, "https://portal.vultisig.com", nil, logger)
		if svc.IsConfigured() {
			t.Error("expected not configured without notification emails")
		}
	})
}

func TestEmailService_SendNotification_NotConfigured(t *testing.T) {
	logger := logrus.New()
	svc := NewEmailService(config.PortalEmailConfig{}, "https://portal.vultisig.com", nil, logger)

	svc.SendProposalNotificationAsync("test-plugin", "Test", "test@example.com")
	svc.SendApprovalNotificationAsync("test-plugin", "Test", "test@example.com")
	svc.SendPublishNotificationAsync("test-plugin", "Test", "test@example.com")
}

func TestEmailService_SendApprovalNotification_EmptyEmail(t *testing.T) {
	logger := logrus.New()
	cfg := config.PortalEmailConfig{
		NotificationEmails: []string{"admin@vultisig.com"},
	}
	svc := NewEmailService(cfg, "https://portal.vultisig.com", nil, logger)

	svc.SendApprovalNotificationAsync("test-plugin", "Test", "")
}

func TestEmailService_SendPublishNotification_EmptyEmail(t *testing.T) {
	logger := logrus.New()
	cfg := config.PortalEmailConfig{
		NotificationEmails: []string{"admin@vultisig.com"},
	}
	svc := NewEmailService(cfg, "https://portal.vultisig.com", nil, logger)

	svc.SendPublishNotificationAsync("test-plugin", "Test", "")
}
