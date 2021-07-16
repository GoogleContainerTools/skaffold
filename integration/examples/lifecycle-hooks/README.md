### Example: Running skaffold lifecycle hooks

This is a simple example to show how to inject skaffold lifecycles with user-defined hooks.

Run:
```
skaffold build --cache-artifacts=false
```

You should see the artifact `hooks-example` being built along with the execution of a `pre-build` hook trigger and a `post-build` hook trigger.

Now with an active kubernetes cluster, run:
```
skaffold dev
```

This will start a pod running the `hooks-example` image. The app simply reads the contents of `hello.txt` file once and stores it and repeatedly prints it out.
```
[hooks-example] Hello World!
[hooks-example] Hello World!
[hooks-example] Hello World!
...
```
If you change the text of `hello.txt` file, say to `Hello World!!!`, Skaffold will `sync` it into the running container.
You should also see a `pre-sync` hook that just echoes the filename that has changed, and a `post-sync` hook that sends a `SIGHUP` signal to the app. The app responds accordingly by reloading the modified `hello.txt` file and printing it to the console. 
```
[hooks-example] Hello World!
[hooks-example] Hello World!
[hooks-example] Hello World!
Syncing 1 files for hooks-example:e3c03bbaf0830afb79d586c5a0d5ce7510a027585fb0cbc88db9bd14bfbad139
Starting pre-sync hooks for artifact "hooks-example"...
file changes detected: hello.txt
Completed pre-sync hooks for artifact "hooks-example"
Starting post-sync hooks for artifact "hooks-example"...
Running command kill -HUP 1
Completed post-sync hooks for artifact "hooks-example"
Watching for changes...
[hooks-example] Hello World!!
[hooks-example] Hello World!!
[hooks-example] Hello World!!
...
```