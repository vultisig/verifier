package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

const defaultMandrillURL = "https://mandrillapp.com/api/1.0/messages/send-template"

type MandrillTo struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
	Type  string `json:"type"`
}

type MandrillMergeVarContent struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type MandrillVar struct {
	Rcpt string                    `json:"rcpt"`
	Vars []MandrillMergeVarContent `json:"vars"`
}

type MandrillMessage struct {
	To            []MandrillTo  `json:"to"`
	SendingDomain string        `json:"sending_domain,omitempty"`
	MergeVars     []MandrillVar `json:"merge_vars,omitempty"`
}

type MandrillPayload struct {
	Key             string                    `json:"key"`
	TemplateName    string                    `json:"template_name"`
	TemplateContent []MandrillMergeVarContent `json:"template_content"`
	Message         MandrillMessage           `json:"message"`
}

type MandrillSendResult struct {
	Email        string `json:"email"`
	Status       string `json:"status"`
	RejectReason string `json:"reject_reason,omitempty"`
	ID           string `json:"_id,omitempty"`
}

type MandrillClient struct {
	apiKey        string
	sendingDomain string
	baseURL       string
	httpClient    *http.Client
	logger        *logrus.Logger
}

func NewMandrillClient(apiKey, sendingDomain string, logger *logrus.Logger) *MandrillClient {
	return &MandrillClient{
		apiKey:        apiKey,
		sendingDomain: sendingDomain,
		baseURL:       defaultMandrillURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func NewMandrillClientWithURL(apiKey, sendingDomain, baseURL string, logger *logrus.Logger) *MandrillClient {
	return &MandrillClient{
		apiKey:        apiKey,
		sendingDomain: sendingDomain,
		baseURL:       baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func (c *MandrillClient) IsConfigured() bool {
	return c.apiKey != ""
}

func (c *MandrillClient) SendTemplate(ctx context.Context, templateName string, recipients []MandrillTo, mergeVars []MandrillVar) error {
	payload := MandrillPayload{
		Key:             c.apiKey,
		TemplateName:    templateName,
		TemplateContent: []MandrillMergeVarContent{},
		Message: MandrillMessage{
			To:            recipients,
			SendingDomain: c.sendingDomain,
			MergeVars:     mergeVars,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mandrill returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var results []MandrillSendResult
	err = json.Unmarshal(respBody, &results)
	if err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	for _, r := range results {
		if r.Status != "sent" && r.Status != "queued" {
			return fmt.Errorf("email to %s failed: %s (%s)", r.Email, r.Status, r.RejectReason)
		}
	}

	return nil
}
