# A2A TASKS GO SDK

![A2A Tasks Go banner](banner.svg)

Reusable Spanner-backed task persistence for `github.com/a2aproject/a2a-go/v2`.

## Package shape

- `SpannerService` owns the raw Spanner schema and CRUD/query logic.
- `SpannerTaskStore` implements `a2asrv/taskstore.Store`.
- `SpannerPushConfigStore` implements `a2asrv/push.ConfigStore`.
- `HTTPPushSender` implements `a2asrv/push.Sender`.

## Usage

`TablePrefix` exists to avoid table-name clashes when multiple services or deployments
share the same Spanner database. The package still uses fixed logical table names
(`Tasks`, `TaskVersions`, `TaskPushConfigs`), but when a prefix is provided they are
resolved as `<prefix>_Tasks`, `<prefix>_TaskVersions`, and
`<prefix>_TaskPushConfigs`.

One practical pattern is to derive the prefix from the project and service name so
each deployment gets its own namespace:

```go
tablePrefix := strings.ReplaceAll(env.MustGet("ALIS_OS_PROJECT"), "-", "_") +
    "_" + strings.ReplaceAll("agent-v2", "-", "_")
```

```go
spannerSvc, err := tasks.NewSpannerService(ctx, tasks.SpannerConfig{
    Project:     projectID,
    Instance:    instanceID,
    Database:    databaseID,
    TablePrefix: tablePrefix,
})
if err != nil {
    return err
}

taskStore := tasks.NewTaskStore(spannerSvc)
pushStore := tasks.NewPushConfigStore(spannerSvc)
sender := tasks.NewHTTPPushSender()
```
