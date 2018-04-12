# Testing

The following guidelines summarize the testing policy for docker/cli.

## Unit Test Suite

All code changes should have unit test coverage.

Error cases should be tested with unit tests.

Bug fixes should be covered by new unit tests or additional assertions in
existing unit tests.

### Details

The unit test suite follows the standard Go testing convention. Tests are
located in the package directory in `_test.go` files.

Unit tests should be named using the convention:

```
Test<Function Name><Test Case Name>
```

[Table tests](https://github.com/golang/go/wiki/TableDrivenTests) should be used
where appropriate, but may not be appropriate in all cases.

Assertions should be made using
[testify/assert](https://godoc.org/github.com/stretchr/testify/assert) and test
requirements should be verified using
[testify/require](https://godoc.org/github.com/stretchr/testify/require).

Fakes, and testing utilities can be found in
[internal/test](https://godoc.org/github.com/docker/cli/internal/test) and
[gotestyourself](https://godoc.org/github.com/gotestyourself/gotestyourself).

## End-to-End Test Suite

The end-to-end test suite tests a cli binary against a real API backend.

### Guidelines

Each feature (subcommand) should have a single end-to-end test for 
the success case. The test should include all (or most) flags/options supported
by that feature.

In some rare cases a couple additional end-to-end tests may be written for a
sufficiently complex and critical feature (ex: `container run`, `service 
create`, `service update`, and `docker build` may have ~3-5 cases each).

In some rare cases a sufficiently critical error paths may have a single
end-to-end test case.

In all other cases the behaviour should be covered by unit tests.

If a code change adds a new flag, that flag should be added to the existing 
"success case" end-to-end test.

If a code change fixes a bug, that bug fix should be covered either by adding 
assertions to the existing end-to-end test, or with one or more unit test.

### Details

The end-to-end test suite is located in
[./e2e](https://github.com/docker/cli/tree/master/e2e). Each directory in `e2e`
corresponds to a directory in `cli/command` and contains the tests for that
subcommand. Files in each directory should be named `<command>_test.go` where
command is the basename of the command (ex: the test for `docker stack deploy`
is found in `e2e/stack/deploy_test.go`).

Tests should be named using the convention:

```
Test<Command Basename>[<Test Case Name>]
```

where the test case name is only required when there are multiple test cases for
a single command.

End-to-end test should run the `docker` binary using
[gotestyourself/icmd](https://godoc.org/github.com/gotestyourself/gotestyourself/icmd)
and make assertions about the exit code, stdout, stderr, and local file system.

Any Docker image or registry operations should use `registry:5000/<image name>`
to communicate with the local instance of the Docker registry. To load 
additional fixture images to the registry see
[scripts/test/e2e/run](https://github.com/docker/cli/blob/master/scripts/test/e2e/run).
