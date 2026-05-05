package portal

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/email"
)

type EmailSender interface {
	IsConfigured() bool
	SendProposalNotificationAsync(pluginID, title, contactEmail string)
	SendApprovalNotificationAsync(pluginID, title, contactEmail string)
	SendPublishNotificationAsync(pluginID, title, contactEmail string)
}

type EmailService struct {
	cfg         config.PortalEmailConfig
	portalURL   string
	queueClient *asynq.Client
	logger      *logrus.Logger
}

func NewEmailService(cfg config.PortalEmailConfig, portalURL string, queueClient *asynq.Client, logger *logrus.Logger) *EmailService {
	return &EmailService{
		cfg:         cfg,
		portalURL:   strings.TrimRight(portalURL, "/"),
		queueClient: queueClient,
		logger:      logger,
	}
}

func (s *EmailService) IsConfigured() bool {
	return s.queueClient != nil && len(s.cfg.NotificationEmails) > 0
}

func (s *EmailService) SendProposalNotificationAsync(pluginID, title, contactEmail string) {
	if !s.IsConfigured() {
		return
	}

	proposalURL := fmt.Sprintf("%s/admin/plugin-proposals/%s", s.portalURL, url.PathEscape(pluginID))

	task := email.PortalProposalTask{
		PluginID:           pluginID,
		Title:              title,
		ContactEmail:       contactEmail,
		ProposalURL:        proposalURL,
		NotificationEmails: s.cfg.NotificationEmails,
	}

	err := s.enqueue(email.TypePortalProposal, task)
	if err != nil {
		s.logger.WithError(err).WithField("plugin_id", pluginID).Error("failed to enqueue proposal email")
	}
}

func (s *EmailService) SendApprovalNotificationAsync(pluginID, title, contactEmail string) {
	if !s.IsConfigured() || contactEmail == "" {
		return
	}

	task := email.PortalApprovalTask{
		PluginID:     pluginID,
		Title:        title,
		ContactEmail: contactEmail,
	}

	err := s.enqueue(email.TypePortalApproval, task)
	if err != nil {
		s.logger.WithError(err).WithField("plugin_id", pluginID).Error("failed to enqueue approval email")
	}
}

func (s *EmailService) SendPublishNotificationAsync(pluginID, title, contactEmail string) {
	if !s.IsConfigured() || contactEmail == "" {
		return
	}

	pluginURL := fmt.Sprintf("%s/plugins/%s", s.portalURL, url.PathEscape(pluginID))

	task := email.PortalPublishTask{
		PluginID:     pluginID,
		Title:        title,
		ContactEmail: contactEmail,
		PluginURL:    pluginURL,
	}

	err := s.enqueue(email.TypePortalPublish, task)
	if err != nil {
		s.logger.WithError(err).WithField("plugin_id", pluginID).Error("failed to enqueue publish email")
	}
}

func (s *EmailService) enqueue(taskType string, payload interface{}) error {
	buf, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}

	_, err = s.queueClient.Enqueue(
		asynq.NewTask(taskType, buf),
		asynq.Queue(email.QueueName),
		asynq.Retention(24*time.Hour),
		asynq.MaxRetry(3),
	)
	if err != nil {
		return fmt.Errorf("enqueue task: %w", err)
	}

	return nil
}
