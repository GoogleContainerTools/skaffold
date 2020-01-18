### Example: Ruby/Rack with hot-reload

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


