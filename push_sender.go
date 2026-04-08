package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv/push"
)

var tokenHeader = http.CanonicalHeaderKey("A2A-Notification-Token")

type HTTPPushSender struct {
	Client *http.Client
}

func NewHTTPPushSender() *HTTPPushSender {
	return &HTTPPushSender{
		Client: &http.Client{Timeout: 30 * time.Second},
	}
}

var _ push.Sender = (*HTTPPushSender)(nil)

func (s *HTTPPushSender) SendPush(ctx context.Context, config *a2a.PushConfig, event a2a.Event) error {
	if config == nil || config.URL == "" {
		return errors.New("config is nil or URL is empty")
	}

	payload, err := pushPayload(event)
	if err != nil {
		return err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal push event: %w", err)
	}

	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create push request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if config.Token != "" {
		req.Header.Set(tokenHeader, config.Token)
	}
	if config.Auth != nil && config.Auth.Credentials != "" {
		switch strings.ToLower(config.Auth.Scheme) {
		case "bearer":
			req.Header.Set("Authorization", "Bearer "+config.Auth.Credentials)
		case "basic":
			req.Header.Set("Authorization", "Basic "+config.Auth.Credentials)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send push notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("push notification endpoint returned non-success status: %s", resp.Status)
	}
	return nil
}

func pushPayload(event a2a.Event) (map[string]any, error) {
	switch v := event.(type) {
	case *a2a.Task:
		return map[string]any{"task": v}, nil
	case *a2a.Message:
		return map[string]any{"message": v}, nil
	case *a2a.TaskStatusUpdateEvent:
		return map[string]any{"statusUpdate": v}, nil
	case *a2a.TaskArtifactUpdateEvent:
		return map[string]any{"artifactUpdate": v}, nil
	default:
		return nil, fmt.Errorf("unknown event type: %T", event)
	}
}
