package email

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
)

func TestHandler_HandleProposal_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"admin@test.com","status":"sent"}]`))
	}))
	defer server.Close()

	logger := logrus.New()
	client := NewMandrillClientWithURL("test-api-key", "test.com", server.URL, logger)
	handler := NewHandler(client, logger)

	task := PortalProposalTask{
		PluginID:           "plugin-123",
		Title:              "Test Plugin",
		ContactEmail:       "dev@example.com",
		ProposalURL:        "https://portal.test.com/admin/plugin-proposals/plugin-123",
		NotificationEmails: []string{"admin@test.com"},
	}
	payload, _ := json.Marshal(task)
	asynqTask := asynq.NewTask(TypePortalProposal, payload)

	err := handler.HandleProposal(context.Background(), asynqTask)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestHandler_HandleProposal_UnmarshalError(t *testing.T) {
	logger := logrus.New()
	client := NewMandrillClient("test-api-key", "test.com", logger)
	handler := NewHandler(client, logger)

	asynqTask := asynq.NewTask(TypePortalProposal, []byte("invalid json"))

	err := handler.HandleProposal(context.Background(), asynqTask)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, asynq.SkipRetry) {
		t.Errorf("expected SkipRetry error, got %v", err)
	}
}

func TestHandler_HandleProposal_MandrillError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status":"error","message":"Internal error"}`))
	}))
	defer server.Close()

	logger := logrus.New()
	client := NewMandrillClientWithURL("test-api-key", "test.com", server.URL, logger)
	handler := NewHandler(client, logger)

	task := PortalProposalTask{
		PluginID:           "plugin-123",
		Title:              "Test Plugin",
		ContactEmail:       "dev@example.com",
		ProposalURL:        "https://portal.test.com/admin/plugin-proposals/plugin-123",
		NotificationEmails: []string{"admin@test.com"},
	}
	payload, _ := json.Marshal(task)
	asynqTask := asynq.NewTask(TypePortalProposal, payload)

	err := handler.HandleProposal(context.Background(), asynqTask)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHandler_HandleApproval_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"dev@example.com","status":"sent"}]`))
	}))
	defer server.Close()

	logger := logrus.New()
	client := NewMandrillClientWithURL("test-api-key", "test.com", server.URL, logger)
	handler := NewHandler(client, logger)

	task := PortalApprovalTask{
		PluginID:     "plugin-123",
		Title:        "Approved Plugin",
		ContactEmail: "dev@example.com",
	}
	payload, _ := json.Marshal(task)
	asynqTask := asynq.NewTask(TypePortalApproval, payload)

	err := handler.HandleApproval(context.Background(), asynqTask)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestHandler_HandleApproval_EmptyContactEmail(t *testing.T) {
	logger := logrus.New()
	client := NewMandrillClient("test-api-key", "test.com", logger)
	handler := NewHandler(client, logger)

	task := PortalApprovalTask{
		PluginID:     "plugin-123",
		Title:        "Test Plugin",
		ContactEmail: "",
	}
	payload, _ := json.Marshal(task)
	asynqTask := asynq.NewTask(TypePortalApproval, payload)

	err := handler.HandleApproval(context.Background(), asynqTask)
	if err != nil {
		t.Errorf("expected no error for empty contact email, got %v", err)
	}
}

func TestHandler_HandleApproval_UnmarshalError(t *testing.T) {
	logger := logrus.New()
	client := NewMandrillClient("test-api-key", "test.com", logger)
	handler := NewHandler(client, logger)

	asynqTask := asynq.NewTask(TypePortalApproval, []byte("invalid json"))

	err := handler.HandleApproval(context.Background(), asynqTask)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, asynq.SkipRetry) {
		t.Errorf("expected SkipRetry error, got %v", err)
	}
}

func TestHandler_HandlePublish_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"dev@example.com","status":"queued"}]`))
	}))
	defer server.Close()

	logger := logrus.New()
	client := NewMandrillClientWithURL("test-api-key", "test.com", server.URL, logger)
	handler := NewHandler(client, logger)

	task := PortalPublishTask{
		PluginID:     "plugin-123",
		Title:        "Published Plugin",
		ContactEmail: "dev@example.com",
		PluginURL:    "https://portal.test.com/plugins/plugin-123",
	}
	payload, _ := json.Marshal(task)
	asynqTask := asynq.NewTask(TypePortalPublish, payload)

	err := handler.HandlePublish(context.Background(), asynqTask)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestHandler_HandlePublish_EmptyContactEmail(t *testing.T) {
	logger := logrus.New()
	client := NewMandrillClient("test-api-key", "test.com", logger)
	handler := NewHandler(client, logger)

	task := PortalPublishTask{
		PluginID:     "plugin-123",
		Title:        "Test Plugin",
		ContactEmail: "",
		PluginURL:    "https://portal.test.com/plugins/plugin-123",
	}
	payload, _ := json.Marshal(task)
	asynqTask := asynq.NewTask(TypePortalPublish, payload)

	err := handler.HandlePublish(context.Background(), asynqTask)
	if err != nil {
		t.Errorf("expected no error for empty contact email, got %v", err)
	}
}

func TestHandler_HandlePublish_UnmarshalError(t *testing.T) {
	logger := logrus.New()
	client := NewMandrillClient("test-api-key", "test.com", logger)
	handler := NewHandler(client, logger)

	asynqTask := asynq.NewTask(TypePortalPublish, []byte("invalid json"))

	err := handler.HandlePublish(context.Background(), asynqTask)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, asynq.SkipRetry) {
		t.Errorf("expected SkipRetry error, got %v", err)
	}
}

func TestHandler_HandlePublish_MandrillReject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"dev@example.com","status":"rejected","reject_reason":"hard-bounce"}]`))
	}))
	defer server.Close()

	logger := logrus.New()
	client := NewMandrillClientWithURL("test-api-key", "test.com", server.URL, logger)
	handler := NewHandler(client, logger)

	task := PortalPublishTask{
		PluginID:     "plugin-123",
		Title:        "Test Plugin",
		ContactEmail: "dev@example.com",
		PluginURL:    "https://portal.test.com/plugins/plugin-123",
	}
	payload, _ := json.Marshal(task)
	asynqTask := asynq.NewTask(TypePortalPublish, payload)

	err := handler.HandlePublish(context.Background(), asynqTask)
	if err == nil {
		t.Fatal("expected error for rejected email, got nil")
	}
}

func TestMandrillClient_IsConfigured(t *testing.T) {
	logger := logrus.New()

	t.Run("configured", func(t *testing.T) {
		client := NewMandrillClient("test-api-key", "test.com", logger)
		if !client.IsConfigured() {
			t.Error("expected client to be configured")
		}
	})

	t.Run("not configured - empty api key", func(t *testing.T) {
		client := NewMandrillClient("", "test.com", logger)
		if client.IsConfigured() {
			t.Error("expected client to not be configured")
		}
	})
}
