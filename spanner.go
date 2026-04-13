package tasks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	tasksTableName                 = "Tasks"
	tasksVersionColumnName         = "latest_version"
	taskVersionsTableName          = "TaskVersions"
	taskVersionsResourceColumnName = "Task"
	taskVersionsUpdatedColumnName  = "last_updated"
	pushConfigsTableName           = "TaskPushConfigs"
	pushConfigsResourceColumnName  = "TaskPushNotificationConfig"
)

type SpannerConfig struct {
	Project      string
	Instance     string
	Database     string
	DatabaseRole string
}

type SpannerService struct {
	db     *spanner.Client
	config SpannerConfig
}

type taskHeadRecord struct {
	TaskID      string
	VersionID   int64
	ContextID   spanner.NullString
	StatusState spanner.NullString
	UpdateTime  spanner.NullTime
}

func NewSpannerService(ctx context.Context, config SpannerConfig) (*SpannerService, error) {
	dbName := fmt.Sprintf("projects/%s/instances/%s/databases/%s", config.Project, config.Instance, config.Database)
	db, err := spanner.NewClientWithConfig(ctx, dbName, spanner.ClientConfig{
		DisableNativeMetrics: true,
		DatabaseRole:         config.DatabaseRole,
	})
	if err != nil {
		return nil, err
	}
	return &SpannerService{db: db, config: config}, nil
}

func NewSpannerServiceWithClient(client *spanner.Client, config SpannerConfig) *SpannerService {
	return &SpannerService{db: client, config: config}
}

func (s *SpannerService) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	s.db.Close()
	return nil
}

func (s *SpannerService) createTask(ctx context.Context, task *TaskProto) (int64, error) {
	if task == nil || task.GetId() == "" {
		return 0, status.Error(codes.InvalidArgument, "task.id is required")
	}
	muts := []*spanner.Mutation{
		spanner.Insert(tasksTableName,
			[]string{"task_id", tasksVersionColumnName},
			[]any{task.GetId(), int64(1)},
		),
		spanner.Insert(taskVersionsTableName,
			[]string{"task_id", "version_id", taskVersionsResourceColumnName},
			[]any{task.GetId(), int64(1), task},
		),
	}
	_, err := s.db.Apply(ctx, muts)
	if spanner.ErrCode(err) == codes.AlreadyExists {
		return 0, status.Error(codes.AlreadyExists, "task already exists")
	}
	return 1, err
}

func (s *SpannerService) readTask(ctx context.Context, taskID string) (*TaskProto, int64, error) {
	stmt := spanner.Statement{
		SQL: fmt.Sprintf(`
SELECT tv.version_id, tv.%s
FROM %s t
JOIN %s tv
  ON t.task_id = tv.task_id AND t.%s = tv.version_id
WHERE t.task_id = @task_id
LIMIT 1`, taskVersionsResourceColumnName, tasksTableName, taskVersionsTableName, tasksVersionColumnName),
		Params: map[string]any{"task_id": taskID},
	}
	iter := s.db.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return nil, 0, status.Error(codes.NotFound, "task not found")
	}
	if err != nil {
		return nil, 0, err
	}
	var version int64
	var task TaskProto
	if err := row.Columns(&version, &task); err != nil {
		return nil, 0, err
	}
	return cloneTaskProto(&task), version, nil
}

func (s *SpannerService) updateTask(ctx context.Context, task *TaskProto, prevVersion int64) (int64, error) {
	if task == nil || task.GetId() == "" {
		return 0, status.Error(codes.InvalidArgument, "task.id is required")
	}

	var nextVersion int64
	_, err := s.db.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, tasksTableName, spanner.Key{task.GetId()}, []string{tasksVersionColumnName})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return status.Error(codes.NotFound, "task not found")
			}
			return err
		}
		var currentVersion int64
		if err := row.Columns(&currentVersion); err != nil {
			return err
		}
		if currentVersion != prevVersion {
			return status.Error(codes.Aborted, "concurrent modification")
		}
		nextVersion = currentVersion + 1
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.Update(tasksTableName,
				[]string{"task_id", tasksVersionColumnName},
				[]any{task.GetId(), nextVersion},
			),
			spanner.Insert(taskVersionsTableName,
				[]string{"task_id", "version_id", taskVersionsResourceColumnName},
				[]any{task.GetId(), nextVersion, task},
			),
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return nextVersion, nil
}

func (s *SpannerService) listTasks(ctx context.Context, req listTasksRequest) ([]*TaskProto, int, string, error) {
	pageSize := normalizePageSize(req.PageSize)
	offset, err := parsePageToken(req.PageToken)
	if err != nil {
		return nil, 0, "", err
	}

	params := map[string]any{
		"limit":  int64(pageSize + 1),
		"offset": int64(offset),
	}
	where, err := s.buildTaskFilter(req, params)
	if err != nil {
		return nil, 0, "", err
	}

	query := fmt.Sprintf(`
SELECT tv.%s
FROM %s t
JOIN %s tv
  ON t.task_id = tv.task_id AND t.%s = tv.version_id`, taskVersionsResourceColumnName, tasksTableName, taskVersionsTableName, tasksVersionColumnName)
	if where != "" {
		query += " WHERE " + where
	}
	query += fmt.Sprintf(" ORDER BY tv.%s DESC, t.task_id ASC LIMIT @limit OFFSET @offset", taskVersionsUpdatedColumnName)

	iter := s.db.Single().Query(ctx, spanner.Statement{SQL: query, Params: params})
	defer iter.Stop()

	var tasks []*TaskProto
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, "", err
		}
		var task TaskProto
		if err := row.Columns(&task); err != nil {
			return nil, 0, "", err
		}
		tasks = append(tasks, cloneTaskProto(&task))
	}

	totalSize, err := s.countTasks(ctx, where, params)
	if err != nil {
		return nil, 0, "", err
	}

	nextPageToken := ""
	if len(tasks) > pageSize {
		tasks = tasks[:pageSize]
		nextPageToken = newPageToken(offset + pageSize)
	}
	return tasks, totalSize, nextPageToken, nil
}

func (s *SpannerService) countTasks(ctx context.Context, where string, params map[string]any) (int, error) {
	countParams := map[string]any{}
	for k, v := range params {
		if k == "limit" || k == "offset" {
			continue
		}
		countParams[k] = v
	}
	query := fmt.Sprintf("SELECT COUNT(1) FROM %s", tasksTableName)
	if where != "" {
		query += " WHERE " + where
	}
	row, err := s.db.Single().Query(ctx, spanner.Statement{SQL: query, Params: countParams}).Next()
	if err != nil {
		return 0, err
	}
	var count int64
	if err := row.Columns(&count); err != nil {
		return 0, err
	}
	return int(count), nil
}

func (s *SpannerService) savePushConfig(ctx context.Context, config *TaskPushConfigProto) error {
	_, err := s.db.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdate(pushConfigsTableName,
			[]string{"config_id", "task_id", pushConfigsResourceColumnName},
			[]any{config.GetId(), config.GetTaskId(), config},
		),
	})
	return err
}

func (s *SpannerService) getPushConfig(ctx context.Context, taskID, configID string) (*TaskPushConfigProto, error) {
	row, err := s.db.Single().ReadRow(ctx, pushConfigsTableName, spanner.Key{configID, taskID}, []string{pushConfigsResourceColumnName})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return nil, status.Error(codes.NotFound, "push config not found")
		}
		return nil, err
	}
	var config TaskPushConfigProto
	if err := row.Columns(&config); err != nil {
		return nil, err
	}
	return clonePushConfigProto(&config), nil
}

func (s *SpannerService) listPushConfigs(ctx context.Context, taskID string) ([]*TaskPushConfigProto, error) {
	stmt := spanner.Statement{
		SQL:    fmt.Sprintf("SELECT %s FROM %s WHERE task_id=@task_id ORDER BY config_id ASC", pushConfigsResourceColumnName, pushConfigsTableName),
		Params: map[string]any{"task_id": taskID},
	}
	iter := s.db.Single().Query(ctx, stmt)
	defer iter.Stop()

	var out []*TaskPushConfigProto
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var config TaskPushConfigProto
		if err := row.Columns(&config); err != nil {
			return nil, err
		}
		out = append(out, clonePushConfigProto(&config))
	}
	return out, nil
}

func (s *SpannerService) deletePushConfig(ctx context.Context, taskID, configID string) error {
	_, err := s.db.Apply(ctx, []*spanner.Mutation{spanner.Delete(pushConfigsTableName, spanner.Key{configID, taskID})})
	return err
}

func (s *SpannerService) deleteAllPushConfigs(ctx context.Context, taskID string) error {
	stmt := spanner.Statement{
		SQL:    fmt.Sprintf("SELECT config_id FROM %s WHERE task_id=@task_id", pushConfigsTableName),
		Params: map[string]any{"task_id": taskID},
	}
	iter := s.db.Single().Query(ctx, stmt)
	defer iter.Stop()

	var muts []*spanner.Mutation
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var configID string
		if err := row.Columns(&configID); err != nil {
			return err
		}
		muts = append(muts, spanner.Delete(pushConfigsTableName, spanner.Key{configID, taskID}))
	}
	if len(muts) == 0 {
		return nil
	}
	_, err := s.db.Apply(ctx, muts)
	return err
}

type listTasksRequest struct {
	ContextID            string
	StatusState          string
	StatusTimestampAfter *time.Time
	PageSize             int
	PageToken            string
}

func (s *SpannerService) buildTaskFilter(req listTasksRequest, params map[string]any) (string, error) {
	var filters []string
	if req.ContextID != "" {
		filters = append(filters, fmt.Sprintf("tv.%s.context_id = @context_id", taskVersionsResourceColumnName))
		params["context_id"] = req.ContextID
	}
	if req.StatusState != "" {
		filters = append(filters, fmt.Sprintf("tv.%s.status.state = @status_state", taskVersionsResourceColumnName))
		params["status_state"] = req.StatusState
	}
	if req.StatusTimestampAfter != nil {
		filters = append(filters, fmt.Sprintf("tv.%s > @status_timestamp_after", taskVersionsUpdatedColumnName))
		params["status_timestamp_after"] = *req.StatusTimestampAfter
	}
	return strings.Join(filters, " AND "), nil
}
