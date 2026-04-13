package tasks

import (
	"context"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv/taskstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SpannerTaskStore struct {
	service *SpannerService
}

func NewTaskStore(service *SpannerService) *SpannerTaskStore {
	return &SpannerTaskStore{service: service}
}

var _ taskstore.Store = (*SpannerTaskStore)(nil)

func (s *SpannerTaskStore) Create(ctx context.Context, task *a2a.Task) (taskstore.TaskVersion, error) {
	protoTask, err := taskToProto(task)
	if err != nil {
		return taskstore.TaskVersionMissing, err
	}
	version, err := s.service.createTask(ctx, protoTask)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return taskstore.TaskVersionMissing, taskstore.ErrTaskAlreadyExists
		}
		return taskstore.TaskVersionMissing, err
	}
	return taskstore.TaskVersion(version), nil
}

func (s *SpannerTaskStore) Update(ctx context.Context, req *taskstore.UpdateRequest) (taskstore.TaskVersion, error) {
	if req == nil || req.Task == nil {
		return taskstore.TaskVersionMissing, status.Error(codes.InvalidArgument, "task is required")
	}
	protoTask, err := taskToProto(req.Task)
	if err != nil {
		return taskstore.TaskVersionMissing, err
	}
	version, err := s.service.updateTask(ctx, protoTask, int64(req.PrevVersion))
	if err != nil {
		switch status.Code(err) {
		case codes.NotFound:
			return taskstore.TaskVersionMissing, a2a.ErrTaskNotFound
		case codes.Aborted:
			return taskstore.TaskVersionMissing, taskstore.ErrConcurrentModification
		default:
			return taskstore.TaskVersionMissing, err
		}
	}
	return taskstore.TaskVersion(version), nil
}

func (s *SpannerTaskStore) Get(ctx context.Context, taskID a2a.TaskID) (*taskstore.StoredTask, error) {
	protoTask, version, err := s.service.readTask(ctx, string(taskID))
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, a2a.ErrTaskNotFound
		}
		return nil, err
	}
	task, err := taskFromProto(protoTask)
	if err != nil {
		return nil, err
	}
	return &taskstore.StoredTask{Task: task, Version: taskstore.TaskVersion(version)}, nil
}

func (s *SpannerTaskStore) List(ctx context.Context, req *a2a.ListTasksRequest) (*a2a.ListTasksResponse, error) {
	if req == nil {
		req = &a2a.ListTasksRequest{}
	}
	protoTasks, totalSize, nextPageToken, err := s.service.listTasks(ctx, listTasksRequest{
		ContextID:            req.ContextID,
		StatusState:          taskStateToProto(req.Status).String(),
		StatusTimestampAfter: req.StatusTimestampAfter,
		PageSize:             req.PageSize,
		PageToken:            req.PageToken,
	})
	if err != nil {
		return nil, err
	}

	tasks := make([]*a2a.Task, 0, len(protoTasks))
	for _, protoTask := range protoTasks {
		task, err := taskFromProto(protoTask)
		if err != nil {
			return nil, err
		}
		if !req.IncludeArtifacts {
			task.Artifacts = nil
		}
		tasks = append(tasks, trimHistory(task, req.HistoryLength))
	}

	return &a2a.ListTasksResponse{
		Tasks:         tasks,
		TotalSize:     totalSize,
		PageSize:      len(tasks),
		NextPageToken: nextPageToken,
	}, nil
}
