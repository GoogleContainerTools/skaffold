### Example: hot-reload with Node and Python

Application demonstrating the file synchronization mode with both NodeJS and Python.

#### Init

```bash
skaffold dev
```

#### Workflow

* Make some changes to `node/src/index.js`:
    * The file will be synchronized to the cluster
    * `nodemon` will restart the application
* Make some changes to `python/src/app.py`:
    * The file will be synchronized to the cluster
    * `flask` will restart the application

<a href="vscode://googlecloudtools.cloudcode/shell?repo=https://github.com/GoogleContainerTools/skaffold.git&subpath=/examples/hot-reload"><img width="286" height="50" src="/docs/static/images/open-cloud-code.png"></a>
