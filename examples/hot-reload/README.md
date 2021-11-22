### Example: hot-reload with Node and Python

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://ssh.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/GoogleContainerTools/skaffold&cloudshell_open_in_editor=README.md&cloudshell_workspace=examples/hot-reload)

Application demonstrating the file synchronization mode with both NodeJS and Python.

#### Init

```bash
skaffold dev [--default-repo docker.io/my-repo]
```

You'll need to specify `--default-repo` to push to a repo you own.
The default value for `--default-repo` is `docker.io/library/`.

#### Workflow

* Make some changes to `node/src/index.js`:
    * The file will be synchronized to the cluster
    * `nodemon` will restart the application
* Make some changes to `python/src/app.py`:
    * The file will be synchronized to the cluster
    * `flask` will restart the application
