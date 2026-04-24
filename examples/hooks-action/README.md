# hooks-action

Demonstrates `deploy.hooks.before` / `after` referencing an existing
custom action via the new `action:` hook form. Running:

```
skaffold deploy --default-repo=gcr.io/my-project
```

executes, in order:

1. `pre-deploy-check` custom action (runs in a busybox container)
2. the kubectl deploy itself
3. `post-deploy-smoke` custom action

Each referenced action must be declared under `customActions`; unknown
names are rejected at config-load time. Action hooks reuse the same
runtime as `skaffold exec`, so `skaffold exec pre-deploy-check` alone
runs the hook standalone.
