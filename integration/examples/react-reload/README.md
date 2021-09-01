### Example: React app with hot-reload

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/react-reload)

Simple React app demonstrating the file synchronization mode in conjunction with webpack hot module reload.

#### Init

```bash
skaffold dev
```

#### Workflow

* Make some changes to `HelloWorld.js`:
    * The file will be synchronized to the cluster
    * `webpack` will perform hot module reloading
* Make some changes to `package.json`:
    * The full build/push/deploy process will be triggered, fetching dependencies from `npm`
