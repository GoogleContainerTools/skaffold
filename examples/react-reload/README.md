### Example: React app with hot-reload

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

<a href="vscode://googlecloudtools.cloudcode/shell?repo=https://github.com/GoogleContainerTools/skaffold.git&subpath=/examples/react-reload"><img width="286" height="50" src="/docs/static/images/open-cloud-code.png"></a>
