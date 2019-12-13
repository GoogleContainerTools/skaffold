### Example: Node.js with hot-reload

Simple example based on Node.js demonstrating the file synchronization mode.

#### Init

```bash
skaffold dev
```

#### Workflow

* Make some changes to `index.js`:
    * The file will be synchronized to the cluster
    * `nodemon` will restart the application
* Make some changes to `package.json`:
    * The full build/push/deploy process will be triggered, fetching dependencies from `npm`


