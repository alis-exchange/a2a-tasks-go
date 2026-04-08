# A2A TASKS GO SDK

![A2A Tasks Go banner](banner.svg)

Reusable Spanner-backed task persistence for `github.com/a2aproject/a2a-go/v2`.

## Package shape

- `SpannerService` owns the raw Spanner schema and CRUD/query logic.
- `SpannerTaskStore` implements `a2asrv/taskstore.Store`.
- `SpannerPushConfigStore` implements `a2asrv/push.ConfigStore`.
- `HTTPPushSender` implements `a2asrv/push.Sender`.

This is intended to replace the current internal package at:

`/Users/jankrynauw/alis.build/alis/build/ge/agent/v2/agent/internal/tasks`

without carrying over direct dependencies on that repo's internal `db`, `auth`,
or `constants` packages.

## Migration direction

Replace app-specific construction like:

```go
store := tasks.NewDatabaseTaskStore(sessionService)
pushStore := tasks.NewDatabasePushConfigService()
sender := tasks.NewHttpPushSender()
```

with:

```go
spannerSvc, err := tasks.NewSpannerService(ctx, tasks.SpannerConfig{
    Project:  projectID,
    Instance: instanceID,
    Database: databaseID,
})
if err != nil {
    return err
}

taskStore := tasks.NewTaskStore(spannerSvc, nil)
pushStore := tasks.NewPushConfigStore(spannerSvc)
sender := tasks.NewHTTPPushSender()
```

If you still need ownership checks, pass an `AccessController` to
`NewTaskStore` instead of hard-coding session/auth logic into the package.
