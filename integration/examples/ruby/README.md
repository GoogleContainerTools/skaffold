### Example: Ruby/Rack with hot-reload

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/ruby)

Simple example based on Ruby/Rack application demonstrating the file synchronization mode.

#### Init

```bash
skaffold dev
```

#### Workflow

* Make some changes to `app.rb`:
    * The file will be synchronized to the cluster
* Make some changes to `Gemfile`:
    * The full build/push/deploy process will be triggered, fetching dependencies from `rubygems`


