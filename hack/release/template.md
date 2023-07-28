# v{{.SkaffoldVersion}} Release - {{.Date}}
**Linux amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v{{.SkaffoldVersion}}/skaffold-linux-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Linux arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v{{.SkaffoldVersion}}/skaffold-linux-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS amd64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v{{.SkaffoldVersion}}/skaffold-darwin-amd64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**macOS arm64**
`curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v{{.SkaffoldVersion}}/skaffold-darwin-arm64 && chmod +x skaffold && sudo mv skaffold /usr/local/bin`

**Windows**
https://storage.googleapis.com/skaffold/releases/v{{.SkaffoldVersion}}/skaffold-windows-amd64.exe

**Docker image**
`gcr.io/k8s-skaffold/skaffold:v{{.SkaffoldVersion}}`
{{.SchemaString}}
Highlights:

New Features and Additions:

Fixes:

Updates and Refactors:

Docs, Test, and Release Updates:

