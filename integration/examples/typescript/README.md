### Example: TypeScript + Node.js with hot-reload

Seeks to be functionally identical to the [Node.js](../nodejs) example, except with TypeScript.

Swaps [nodemon](https://nodemon.io/) for [tsc-watch](https://github.com/gilamran/tsc-watch#the-nodemon-for-typescript)

#### Init

```bash
skaffold dev
```

#### Workflow

* Make some changes to [index.ts](./backend/src/index.ts):
    * The file will be synchronized to the cluster
    * `tsc-watch` will restart the application
* Make some changes to `package.json`:
    * The full build/push/deploy process will be triggered, fetching dependencies from `npm`
* NOTE: Currently in this example, `tsc-watch` is not configured when `ENV=production`.  To configure it, update the [package.json](./backend/src/package.json) file `scripts.production"` field to be:
  * `"production": "tsc && node ./src/index.js"` instead of the default `"production": "node src/index.js",`
  
