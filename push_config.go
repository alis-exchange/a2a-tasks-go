package tasks

import (
	"context"
	"fmt"
	"net/url"

	"github.com/a2aproject/a2a-go/v2/a2a"
	a2apb "github.com/a2aproject/a2a-go/v2/a2apb/v1"
	"github.com/a2aproject/a2a-go/v2/a2asrv/push"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SpannerPushConfigStore struct {
	service *SpannerService
}

func NewPushConfigStore(service *SpannerService) *SpannerPushConfigStore {
	return &SpannerPushConfigStore{service: service}
}

var _ push.ConfigStore = (*SpannerPushConfigStore)(nil)

func (s *SpannerPushConfigStore) Save(ctx context.Context, taskID a2a.TaskID, config *a2a.PushConfig) (*a2a.PushConfig, error) {
	if err := validatePushConfig(config); err != nil {
		return nil, fmt.Errorf("%w: %w", a2a.ErrInvalidParams, err)
	}

	configID := config.ID
	if configID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return nil, err
		}
		configID = id.String()
	}

	protoConfig := &TaskPushConfigProto{
		Id:     configID,
		TaskId: string(taskID),
		Url:    config.URL,
		Token:  config.Token,
	}
	if config.Auth != nil {
		protoConfig.Authentication = &a2apb.AuthenticationInfo{
			Credentials: config.Auth.Credentials,
			Scheme:      config.Auth.Scheme,
		}
	}
	if err := s.service.savePushConfig(ctx, protoConfig); err != nil {
		return nil, err
	}

	saved := *config
	saved.ID = configID
	return &saved, nil
}

func (s *SpannerPushConfigStore) Get(ctx context.Context, taskID a2a.TaskID, configID string) (*a2a.PushConfig, error) {
	protoConfig, err := s.service.getPushConfig(ctx, string(taskID), configID)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, push.ErrPushConfigNotFound
		}
		return nil, err
	}
	return pushConfigFromProto(protoConfig), nil
}

func (s *SpannerPushConfigStore) List(ctx context.Context, taskID a2a.TaskID) ([]*a2a.PushConfig, error) {
	protoConfigs, err := s.service.listPushConfigs(ctx, string(taskID))
	if err != nil {
		return nil, err
	}
	out := make([]*a2a.PushConfig, len(protoConfigs))
	for i, config := range protoConfigs {
		out[i] = pushConfigFromProto(config)
	}
	return out, nil
}

func (s *SpannerPushConfigStore) Delete(ctx context.Context, taskID a2a.TaskID, configID string) error {
	return s.service.deletePushConfig(ctx, string(taskID), configID)
}

func (s *SpannerPushConfigStore) DeleteAll(ctx context.Context, taskID a2a.TaskID) error {
	return s.service.deleteAllPushConfigs(ctx, string(taskID))
}

func pushConfigFromProto(in *TaskPushConfigProto) *a2a.PushConfig {
	if in == nil {
		return nil
	}
	out := &a2a.PushConfig{
		ID:    in.GetId(),
		Token: in.GetToken(),
		URL:   in.GetUrl(),
	}
	if auth := in.GetAuthentication(); auth != nil {
		out.Auth = &a2a.PushAuthInfo{
			Credentials: auth.GetCredentials(),
			Scheme:      auth.GetScheme(),
		}
	}
	return out
}

func validatePushConfig(config *a2a.PushConfig) error {
	if config == nil {
		return fmt.Errorf("push config cannot be nil")
	}
	if config.URL == "" {
		return fmt.Errorf("push config endpoint cannot be empty")
	}
	if _, err := url.ParseRequestURI(config.URL); err != nil {
		return fmt.Errorf("invalid push config endpoint URL: %w", err)
	}
	return nil
}
