# Licenses tool

This tool analyzes the dependency tree of a Go package/binary. It can output a report on the libraries used and under what license they can be used. It can also collect all of the license documents, copyright notices and source code into a directory in order to comply with license terms on redistribution.

## Reports

```shell
$ licenses csv "github.com/google/trillian/server/trillian_log_server"
google.golang.org/grpc,https://github.com/grpc/grpc-go/blob/master/LICENSE,Apache-2.0
go.opencensus.io,https://github.com/census-instrumentation/opencensus-go/blob/master/LICENSE,Apache-2.0
github.com/google/certificate-transparency-go,https://github.com/google/certificate-transparency-go/blob/master/LICENSE,Apache-2.0
github.com/jmespath/go-jmespath,https://github.com/aws/aws-sdk-go/blob/master/vendor/github.com/jmespath/go-jmespath/LICENSE,Apache-2.0
golang.org/x/text,https://go.googlesource.com/text/+/refs/heads/master/LICENSE,BSD-3-Clause
golang.org/x/sync/semaphore,https://go.googlesource.com/sync/+/refs/heads/master/LICENSE,BSD-3-Clause
github.com/prometheus/client_model/go,https://github.com/prometheus/client_model/blob/master/LICENSE,Apache-2.0
github.com/beorn7/perks/quantile,https://github.com/beorn7/perks/blob/master/LICENSE,MIT
```

This command prints out a comma-separated report (CSV) listing the libraries used by a binary/package, the URL where their licenses can be viewed and the type of license. A library is considered to be one or more Go packages that share a license file.

URLs will not be available if the library is not checked out as a Git repository (e.g. as is the case when Go Modules are enabled).

## Complying with license terms

```shell
$ licenses save "github.com/google/trillian/server/trillian_log_server" --save_dir="/tmp/trillian_log_server"
```

This command analyzes a binary/package's dependencies and determines what needs to be redistributed alongside that binary/package in order to comply with the license terms. This typically includes the license itself and a copyright notice, but may also include the dependency's source code. All of the required artifacts will be saved in the directory indicated by `--save_dir`.

## Warnings and errors

The tool will log warnings and errors in some scenarios. This section provides guidance on addressing them.

### Dependency contains non-Go code

A warning will be logged when a dependency contains non-Go code. This is because it is not possible to check the non-Go code for further dependencies, which may conceal additional license requirements. You should investigate this code to determine whether it has dependencies and take action to comply with their license terms.

### Error discovering URL

In order to determine the URL where a license file can be viewed, this tool performs the following steps:

1) Locates the license file on disk.
2) Assuming that it is in a Git repository, inspects the repository's config to find the URL of the remote "origin" repository.
3) Adds the license file path to this URL.

For this to work, the remote repository named "origin" must have a HTTPS URL. You can check this by running the following commands,
inserting the path mentioned in the log message:

```shell
$ cd "path/mentioned/in/log/message"
$ git remote get-url origin
https://github.com/google/trillian.git
```

If you want the tool to use a different remote repository, use the `--git_remote` flag. You can pass this flag repeatedly to make the tool try a number of different remotes.