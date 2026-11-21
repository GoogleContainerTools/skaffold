# Custom actions: deploy parameters

`skaffold exec` forwards deploy parameters — values from `--set` or
`--set-value-file` — as environment variables into every container of the
invoked custom action, mirroring [Google Cloud Deploy](https://docs.cloud.google.com/deploy/docs/parameters).

```console
$ skaffold exec show-params --set TF_VAR_bucket=my-bkt --set REGION=us-central1
```

The `printer` container echoes `TF_VAR_bucket=my-bkt` and `REGION=us-central1`.
Precedence (lowest to highest): base env / `--env-file` < `--set-value-file` < `--set`.
On key collision, deploy parameters override a container's own `env:` entry.
