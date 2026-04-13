// Package tasks provides reusable Spanner-backed persistence for A2A tasks and
// task push notification configuration.
//
// The package intentionally separates storage from protocol-facing adapters:
//
//   - SpannerService owns the physical schema, querying and mutation logic.
//   - SpannerTaskStore implements github.com/a2aproject/a2a-go/v2/a2asrv/taskstore.Store.
//   - SpannerPushConfigStore implements github.com/a2aproject/a2a-go/v2/a2asrv/push.ConfigStore.
//
// SpannerConfig.TablePrefix can be used to namespace the physical table names when
// multiple services or deployments share one Spanner database. For example, a
// prefix of "my_agent" resolves the built-in table names to "my_agent_Tasks",
// "my_agent_TaskVersions", and "my_agent_TaskPushConfigs".
//
// Expected Spanner schema:
//
// Tasks table:
//
//	task_id STRING(MAX) NOT NULL,
//	latest_version INT64 NOT NULL,
//
// TaskVersions table:
//
//	task_id STRING(MAX) NOT NULL,
//	version_id INT64 NOT NULL,
//	Task PROTO<a2a.v1.Task> NOT NULL,
//	Policy PROTO<google.iam.v1.Policy>,
//	last_updated TIMESTAMP AS (...) STORED,
//
// TaskVersions is interleaved in Tasks with ON DELETE CASCADE.
//
// TaskPushConfigs table:
//
//	config_id STRING(MAX) NOT NULL,
//	task_id STRING(MAX) NOT NULL,
//	TaskPushNotificationConfig PROTO<a2a.v1.TaskPushNotificationConfig> NOT NULL,
//	Policy PROTO<google.iam.v1.Policy>
package tasks
