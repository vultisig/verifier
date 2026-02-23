package email

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	client *MandrillClient
	logger *logrus.Logger
}

func NewHandler(client *MandrillClient, logger *logrus.Logger) *Handler {
	return &Handler{
		client: client,
		logger: logger,
	}
}

func (h *Handler) HandleProposal(ctx context.Context, t *asynq.Task) error {
	var task PortalProposalTask
	err := json.Unmarshal(t.Payload(), &task)
	if err != nil {
		return fmt.Errorf("unmarshal task: %w: %w", err, asynq.SkipRetry)
	}

	recipients := make([]MandrillTo, len(task.NotificationEmails))
	for i, addr := range task.NotificationEmails {
		recipients[i] = MandrillTo{
			Email: addr,
			Type:  "to",
		}
	}

	mergeVars := make([]MandrillVar, len(task.NotificationEmails))
	for i, addr := range task.NotificationEmails {
		mergeVars[i] = MandrillVar{
			Rcpt: addr,
			Vars: []MandrillMergeVarContent{
				{Name: MergeVarPluginID, Content: task.PluginID},
				{Name: MergeVarPluginTitle, Content: task.Title},
				{Name: MergeVarContactEmail, Content: task.ContactEmail},
				{Name: MergeVarProposalURL, Content: task.ProposalURL},
			},
		}
	}

	err = h.client.SendTemplate(ctx, TemplatePortalProposal, recipients, mergeVars)
	if err != nil {
		h.logger.WithError(err).WithField("plugin_id", task.PluginID).Error("failed to send proposal email")
		return fmt.Errorf("send template: %w", err)
	}

	h.logger.WithField("plugin_id", task.PluginID).Info("proposal notification sent")
	return nil
}

func (h *Handler) HandleApproval(ctx context.Context, t *asynq.Task) error {
	var task PortalApprovalTask
	err := json.Unmarshal(t.Payload(), &task)
	if err != nil {
		return fmt.Errorf("unmarshal task: %w: %w", err, asynq.SkipRetry)
	}

	return h.sendContactEmail(ctx, TemplatePortalApproval, task.ContactEmail, task.PluginID, []MandrillMergeVarContent{
		{Name: MergeVarPluginID, Content: task.PluginID},
		{Name: MergeVarPluginTitle, Content: task.Title},
	})
}

func (h *Handler) HandlePublish(ctx context.Context, t *asynq.Task) error {
	var task PortalPublishTask
	err := json.Unmarshal(t.Payload(), &task)
	if err != nil {
		return fmt.Errorf("unmarshal task: %w: %w", err, asynq.SkipRetry)
	}

	return h.sendContactEmail(ctx, TemplatePortalPublish, task.ContactEmail, task.PluginID, []MandrillMergeVarContent{
		{Name: MergeVarPluginID, Content: task.PluginID},
		{Name: MergeVarPluginTitle, Content: task.Title},
		{Name: MergeVarPluginURL, Content: task.PluginURL},
	})
}

func (h *Handler) sendContactEmail(ctx context.Context, template string, contactEmail string, pluginID string, vars []MandrillMergeVarContent) error {
	if contactEmail == "" {
		return nil
	}

	recipients := []MandrillTo{
		{Email: contactEmail, Type: "to"},
	}

	mergeVars := []MandrillVar{
		{Rcpt: contactEmail, Vars: vars},
	}

	err := h.client.SendTemplate(ctx, template, recipients, mergeVars)
	if err != nil {
		h.logger.WithError(err).WithField("plugin_id", pluginID).Errorf("failed to send %s email", template)
		return fmt.Errorf("send template: %w", err)
	}

	h.logger.WithField("plugin_id", pluginID).Infof("%s notification sent", template)
	return nil
}
