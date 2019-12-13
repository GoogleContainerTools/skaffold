### Example: Getting started with a simple go app

This is a simple show-case of how Skaffold profiles can be patched.
Patched profiles e.g. can be used in development to provide a composable development setup.
Here, a "base" service is always started. Two additional services "hello" and "world" can be activated via profiles.

#### Init

Use the `--profile` option to add profiles `skaffold dev --profile hello,world`

#### Workflow

* Build only the `base-service` when using the main profile
* Build `hello` and/or `world` when specified via `-p` flag. Multiple `-p` flags are supported as well as comma separated values.