package tasks

import (
	"fmt"

	"github.com/a2aproject/a2a-go/v2/a2a"
	a2apb "github.com/a2aproject/a2a-go/v2/a2apb/v1"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type (
	TaskProto           = a2apb.Task
	TaskPushConfigProto = a2apb.TaskPushNotificationConfig
)

func taskToProto(task *a2a.Task) (*TaskProto, error) {
	if task == nil {
		return nil, nil
	}

	status, err := taskStatusToProto(task.Status)
	if err != nil {
		return nil, fmt.Errorf("convert task status: %w", err)
	}
	artifacts, err := artifactsToProto(task.Artifacts)
	if err != nil {
		return nil, fmt.Errorf("convert artifacts: %w", err)
	}
	history, err := messagesToProto(task.History)
	if err != nil {
		return nil, fmt.Errorf("convert history: %w", err)
	}
	metadata, err := mapToProto(task.Metadata)
	if err != nil {
		return nil, fmt.Errorf("convert task metadata: %w", err)
	}

	return &a2apb.Task{
		Id:        string(task.ID),
		ContextId: task.ContextID,
		Status:    status,
		Artifacts: artifacts,
		History:   history,
		Metadata:  metadata,
	}, nil
}

func taskFromProto(task *TaskProto) (*a2a.Task, error) {
	if task == nil {
		return nil, nil
	}

	status, err := taskStatusFromProto(task.GetStatus())
	if err != nil {
		return nil, fmt.Errorf("convert task status: %w", err)
	}
	artifacts, err := artifactsFromProto(task.GetArtifacts())
	if err != nil {
		return nil, fmt.Errorf("convert artifacts: %w", err)
	}
	history, err := messagesFromProto(task.GetHistory())
	if err != nil {
		return nil, fmt.Errorf("convert history: %w", err)
	}

	return &a2a.Task{
		ID:        a2a.TaskID(task.GetId()),
		ContextID: task.GetContextId(),
		Status:    status,
		Artifacts: artifacts,
		History:   history,
		Metadata:  mapFromProto(task.GetMetadata()),
	}, nil
}

func messagesToProto(in []*a2a.Message) ([]*a2apb.Message, error) {
	out := make([]*a2apb.Message, len(in))
	for i, msg := range in {
		converted, err := messageToProto(msg)
		if err != nil {
			return nil, err
		}
		out[i] = converted
	}
	return out, nil
}

func messagesFromProto(in []*a2apb.Message) ([]*a2a.Message, error) {
	out := make([]*a2a.Message, len(in))
	for i, msg := range in {
		converted, err := messageFromProto(msg)
		if err != nil {
			return nil, err
		}
		out[i] = converted
	}
	return out, nil
}

func messageToProto(msg *a2a.Message) (*a2apb.Message, error) {
	if msg == nil {
		return nil, nil
	}
	parts, err := partsToProto(msg.Parts)
	if err != nil {
		return nil, err
	}
	meta, err := mapToProto(msg.Metadata)
	if err != nil {
		return nil, err
	}

	taskIDs := make([]string, len(msg.ReferenceTasks))
	for i, id := range msg.ReferenceTasks {
		taskIDs[i] = string(id)
	}

	return &a2apb.Message{
		MessageId:        msg.ID,
		ContextId:        msg.ContextID,
		TaskId:           string(msg.TaskID),
		Role:             roleToProto(msg.Role),
		Parts:            parts,
		Metadata:         meta,
		Extensions:       msg.Extensions,
		ReferenceTaskIds: taskIDs,
	}, nil
}

func messageFromProto(msg *a2apb.Message) (*a2a.Message, error) {
	if msg == nil {
		return nil, nil
	}
	parts, err := partsFromProto(msg.GetParts())
	if err != nil {
		return nil, err
	}

	out := &a2a.Message{
		ID:         msg.GetMessageId(),
		ContextID:  msg.GetContextId(),
		TaskID:     a2a.TaskID(msg.GetTaskId()),
		Role:       roleFromProto(msg.GetRole()),
		Parts:      parts,
		Metadata:   mapFromProto(msg.GetMetadata()),
		Extensions: msg.GetExtensions(),
	}
	if refs := msg.GetReferenceTaskIds(); len(refs) > 0 {
		out.ReferenceTasks = make([]a2a.TaskID, len(refs))
		for i, id := range refs {
			out.ReferenceTasks[i] = a2a.TaskID(id)
		}
	}
	return out, nil
}

func artifactsToProto(in []*a2a.Artifact) ([]*a2apb.Artifact, error) {
	out := make([]*a2apb.Artifact, len(in))
	for i, artifact := range in {
		converted, err := artifactToProto(artifact)
		if err != nil {
			return nil, err
		}
		out[i] = converted
	}
	return out, nil
}

func artifactsFromProto(in []*a2apb.Artifact) ([]*a2a.Artifact, error) {
	out := make([]*a2a.Artifact, len(in))
	for i, artifact := range in {
		converted, err := artifactFromProto(artifact)
		if err != nil {
			return nil, err
		}
		out[i] = converted
	}
	return out, nil
}

func artifactToProto(in *a2a.Artifact) (*a2apb.Artifact, error) {
	if in == nil {
		return nil, nil
	}
	parts, err := partsToProto(in.Parts)
	if err != nil {
		return nil, err
	}
	meta, err := mapToProto(in.Metadata)
	if err != nil {
		return nil, err
	}
	return &a2apb.Artifact{
		ArtifactId:  string(in.ID),
		Name:        in.Name,
		Description: in.Description,
		Parts:       parts,
		Metadata:    meta,
		Extensions:  in.Extensions,
	}, nil
}

func artifactFromProto(in *a2apb.Artifact) (*a2a.Artifact, error) {
	if in == nil {
		return nil, nil
	}
	parts, err := partsFromProto(in.GetParts())
	if err != nil {
		return nil, err
	}
	return &a2a.Artifact{
		ID:          a2a.ArtifactID(in.GetArtifactId()),
		Name:        in.GetName(),
		Description: in.GetDescription(),
		Parts:       parts,
		Metadata:    mapFromProto(in.GetMetadata()),
		Extensions:  in.GetExtensions(),
	}, nil
}

func partsToProto(in []*a2a.Part) ([]*a2apb.Part, error) {
	out := make([]*a2apb.Part, len(in))
	for i, part := range in {
		converted, err := partToProto(part)
		if err != nil {
			return nil, err
		}
		out[i] = converted
	}
	return out, nil
}

func partsFromProto(in []*a2apb.Part) ([]*a2a.Part, error) {
	out := make([]*a2a.Part, len(in))
	for i, part := range in {
		converted, err := partFromProto(part)
		if err != nil {
			return nil, err
		}
		out[i] = converted
	}
	return out, nil
}

func partToProto(in *a2a.Part) (*a2apb.Part, error) {
	if in == nil {
		return nil, nil
	}
	meta, err := mapToProto(in.Metadata)
	if err != nil {
		return nil, err
	}
	out := &a2apb.Part{
		Metadata:  meta,
		Filename:  in.Filename,
		MediaType: in.MediaType,
	}
	switch v := in.Content.(type) {
	case a2a.Text:
		out.Content = &a2apb.Part_Text{Text: string(v)}
	case a2a.Data:
		// If the value is nil, create a new map[string]any
		val := v.Value
		if val == nil {
			val = make(map[string]any)
		}

		value, err := structpb.NewValue(val)
		if err != nil {
			return nil, err
		}

		out.Content = &a2apb.Part_Data{Data: value}
	case a2a.URL:
		out.Content = &a2apb.Part_Url{Url: string(v)}
	case a2a.Raw:
		out.Content = &a2apb.Part_Raw{Raw: []byte(v)}
	default:
		return nil, fmt.Errorf("unsupported part type %T", in.Content)
	}
	return out, nil
}

func partFromProto(in *a2apb.Part) (*a2a.Part, error) {
	if in == nil {
		return nil, nil
	}
	meta := mapFromProto(in.GetMetadata())

	var part *a2a.Part
	switch v := in.GetContent().(type) {
	case *a2apb.Part_Text:
		part = a2a.NewTextPart(v.Text)
	case *a2apb.Part_Data:
		part = a2a.NewDataPart(v.Data.AsInterface())
	case *a2apb.Part_Url:
		part = a2a.NewFileURLPart(a2a.URL(v.Url), in.GetMediaType())
	case *a2apb.Part_Raw:
		part = a2a.NewRawPart(v.Raw)
	default:
		return nil, fmt.Errorf("unsupported proto part type %T", in.GetContent())
	}

	part.Filename = in.GetFilename()
	part.MediaType = in.GetMediaType()
	for k, v := range meta {
		part.SetMeta(k, v)
	}
	return part, nil
}

func taskStatusToProto(in a2a.TaskStatus) (*a2apb.TaskStatus, error) {
	msg, err := messageToProto(in.Message)
	if err != nil {
		return nil, err
	}
	out := &a2apb.TaskStatus{
		State:   taskStateToProto(in.State),
		Message: msg,
	}
	if in.Timestamp != nil {
		out.Timestamp = timestamppb.New(*in.Timestamp)
	}
	return out, nil
}

func taskStatusFromProto(in *a2apb.TaskStatus) (a2a.TaskStatus, error) {
	if in == nil {
		return a2a.TaskStatus{}, fmt.Errorf("task status is required")
	}
	msg, err := messageFromProto(in.GetMessage())
	if err != nil {
		return a2a.TaskStatus{}, err
	}
	out := a2a.TaskStatus{
		State:   taskStateFromProto(in.GetState()),
		Message: msg,
	}
	if ts := in.GetTimestamp(); ts != nil {
		t := ts.AsTime()
		out.Timestamp = &t
	}
	return out, nil
}

func mapToProto(in map[string]any) (*structpb.Struct, error) {
	if in == nil {
		return nil, nil
	}
	return structpb.NewStruct(in)
}

func mapFromProto(in *structpb.Struct) map[string]any {
	if in == nil {
		return nil
	}
	return in.AsMap()
}

func roleToProto(in a2a.MessageRole) a2apb.Role {
	switch in {
	case a2a.MessageRoleUser:
		return a2apb.Role_ROLE_USER
	case a2a.MessageRoleAgent:
		return a2apb.Role_ROLE_AGENT
	default:
		return a2apb.Role_ROLE_UNSPECIFIED
	}
}

func roleFromProto(in a2apb.Role) a2a.MessageRole {
	switch in {
	case a2apb.Role_ROLE_USER:
		return a2a.MessageRoleUser
	case a2apb.Role_ROLE_AGENT:
		return a2a.MessageRoleAgent
	default:
		return a2a.MessageRoleUnspecified
	}
}

func taskStateToProto(in a2a.TaskState) a2apb.TaskState {
	switch in {
	case a2a.TaskStateSubmitted:
		return a2apb.TaskState_TASK_STATE_SUBMITTED
	case a2a.TaskStateWorking:
		return a2apb.TaskState_TASK_STATE_WORKING
	case a2a.TaskStateCompleted:
		return a2apb.TaskState_TASK_STATE_COMPLETED
	case a2a.TaskStateFailed:
		return a2apb.TaskState_TASK_STATE_FAILED
	case a2a.TaskStateCanceled:
		return a2apb.TaskState_TASK_STATE_CANCELED
	case a2a.TaskStateInputRequired:
		return a2apb.TaskState_TASK_STATE_INPUT_REQUIRED
	case a2a.TaskStateRejected:
		return a2apb.TaskState_TASK_STATE_REJECTED
	case a2a.TaskStateAuthRequired:
		return a2apb.TaskState_TASK_STATE_AUTH_REQUIRED
	default:
		return a2apb.TaskState_TASK_STATE_UNSPECIFIED
	}
}

func taskStateFromProto(in a2apb.TaskState) a2a.TaskState {
	switch in {
	case a2apb.TaskState_TASK_STATE_SUBMITTED:
		return a2a.TaskStateSubmitted
	case a2apb.TaskState_TASK_STATE_WORKING:
		return a2a.TaskStateWorking
	case a2apb.TaskState_TASK_STATE_COMPLETED:
		return a2a.TaskStateCompleted
	case a2apb.TaskState_TASK_STATE_FAILED:
		return a2a.TaskStateFailed
	case a2apb.TaskState_TASK_STATE_CANCELED:
		return a2a.TaskStateCanceled
	case a2apb.TaskState_TASK_STATE_INPUT_REQUIRED:
		return a2a.TaskStateInputRequired
	case a2apb.TaskState_TASK_STATE_REJECTED:
		return a2a.TaskStateRejected
	case a2apb.TaskState_TASK_STATE_AUTH_REQUIRED:
		return a2a.TaskStateAuthRequired
	default:
		return a2a.TaskStateUnspecified
	}
}
