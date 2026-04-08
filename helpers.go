package tasks

import (
	"encoding/base64"
	"strconv"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func cloneTaskProto(in *TaskProto) *TaskProto {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*TaskProto)
}

func clonePushConfigProto(in *TaskPushConfigProto) *TaskPushConfigProto {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*TaskPushConfigProto)
}

func newPageToken(offset int) string {
	if offset <= 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func parsePageToken(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	raw, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return 0, status.Error(codes.InvalidArgument, "invalid page_token")
	}
	offset, err := strconv.Atoi(string(raw))
	if err != nil || offset < 0 {
		return 0, status.Error(codes.InvalidArgument, "invalid page_token")
	}
	return offset, nil
}

func normalizePageSize(size int) int {
	switch {
	case size <= 0:
		return 50
	case size > 100:
		return 100
	default:
		return size
	}
}

func trimHistory(task *a2a.Task, historyLength *int) *a2a.Task {
	if task == nil || historyLength == nil || *historyLength < 0 || len(task.History) <= *historyLength {
		return task
	}
	clone := *task
	clone.History = append([]*a2a.Message(nil), task.History[:*historyLength]...)
	return &clone
}
