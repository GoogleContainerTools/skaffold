package cmd

import (
	"fmt"
	"os"
	"strings"
)

const (
	// lifecycle errors not specific to any phase: 1-99
	CodeFailed = 1 // CodeFailed indicates generic lifecycle error
	// 2: reserved
	CodeInvalidArgs = 3
	// 4: CodeInvalidEnv
	// 5: CodeNotFound
	// 9: CodeFailedUpdate

	// API errors
	CodeIncompatiblePlatformAPI  = 11
	CodeIncompatibleBuildpackAPI = 12

	// detect phase errors: 100-199
	CodeFailedDetect = 100 // CodeFailedDetect indicates that no buildpacks detected
	// CodeFailedDetectWithErrors indicated that no buildpacks detected and at least one errored
	CodeFailedDetectWithErrors = 101
	CodeDetectError            = 102 // CodeDetectError indicates generic detect error

	// analyze phase errors: 200-299
	CodeAnalyzeError = 202 // CodeAnalyzeError indicates generic analyze error

	// restore phase errors: 300-399
	CodeRestoreError = 302 // CodeRestoreError indicates generic restore error

	// build phase errors: 400-499
	CodeFailedBuildWithErrors = 401 // CodeFailedBuildWithErrors indicates buildpack error during /bin/build
	CodeBuildError            = 402 // CodeBuildError indicates generic build error

	// export phase errors: 500-599
	CodeExportError = 502 // CodeExportError indicates generic export error

	// rebase phase errors: 600-699
	CodeRebaseError = 602 // CodeRebaseError indicates generic rebase error

	// launch phase errors: 700-799
	CodeLaunchError = 702 // CodeLaunchError indicates generic launch error
)

type ErrorFail struct {
	Err    error
	Code   int
	Action []string
}

func (e *ErrorFail) Error() string {
	message := "failed to " + strings.Join(e.Action, " ")
	if e.Err == nil {
		return message
	}
	return fmt.Sprintf("%s: %s", message, e.Err)
}

func FailCode(code int, action ...string) *ErrorFail {
	return FailErrCode(nil, code, action...)
}

func FailErr(err error, action ...string) *ErrorFail {
	code := CodeFailed
	if err, ok := err.(*ErrorFail); ok {
		code = err.Code
	}
	return FailErrCode(err, code, action...)
}

func FailErrCode(err error, code int, action ...string) *ErrorFail {
	return &ErrorFail{Err: err, Code: code, Action: action}
}

func Exit(err error) {
	if err == nil {
		os.Exit(0)
	}
	DefaultLogger.Errorf("%s\n", err)
	if err, ok := err.(*ErrorFail); ok {
		os.Exit(err.Code)
	}
	os.Exit(CodeFailed)
}

func ExitWithVersion() {
	DefaultLogger.Infof(buildVersion())
	os.Exit(0)
}
