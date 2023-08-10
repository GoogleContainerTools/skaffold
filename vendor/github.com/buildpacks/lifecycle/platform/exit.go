package platform

type LifecycleExitError int

const CodeFailed = 1

const (
	FailedDetect           LifecycleExitError = iota
	FailedDetectWithErrors                    // no buildpacks detected
	DetectError                               // no buildpacks detected and at least one errored
	AnalyzeError                              // generic analyze error
	RestoreError                              // generic restore error
	FailedBuildWithErrors                     // buildpack error during /bin/build
	BuildError                                // generic build error
	ExportError                               // generic export error
	RebaseError                               // generic rebase error
	LaunchError                               // generic launch error
)

type Exiter interface {
	CodeFor(errType LifecycleExitError) int
}

type DefaultExiter struct{}

var defaultExitCodes = map[LifecycleExitError]int{
	// detect phase errors: 20-29
	FailedDetect:           20, // FailedDetect indicates that no buildpacks detected
	FailedDetectWithErrors: 21, // FailedDetectWithErrors indicated that no buildpacks detected and at least one errored
	DetectError:            22, // DetectError indicates generic detect error

	// analyze phase errors: 30-39
	AnalyzeError: 32, // AnalyzeError indicates generic analyze error

	// restore phase errors: 40-49
	RestoreError: 42, // RestoreError indicates generic restore error

	// build phase errors: 50-59
	FailedBuildWithErrors: 51, // FailedBuildWithErrors indicates buildpack error during /bin/build
	BuildError:            52, // BuildError indicates generic build error

	// export phase errors: 60-69
	ExportError: 62, // ExportError indicates generic export error

	// rebase phase errors: 70-79
	RebaseError: 72, // RebaseError indicates generic rebase error

	// launch phase errors: 80-89
	LaunchError: 82, // LaunchError indicates generic launch error
}

func (e *DefaultExiter) CodeFor(errType LifecycleExitError) int {
	return codeFor(errType, defaultExitCodes)
}

type LegacyExiter struct{}

var legacyExitCodes = map[LifecycleExitError]int{
	// detect phase errors: 100-199
	FailedDetect:           100, // FailedDetect indicates that no buildpacks detected
	FailedDetectWithErrors: 101, // FailedDetectWithErrors indicated that no buildpacks detected and at least one errored
	DetectError:            102, // DetectError indicates generic detect error

	// analyze phase errors: 200-299
	AnalyzeError: 202, // AnalyzeError indicates generic analyze error

	// restore phase errors: 300-399
	RestoreError: 302, // RestoreError indicates generic restore error

	// build phase errors: 400-499
	FailedBuildWithErrors: 401, // FailedBuildWithErrors indicates buildpack error during /bin/build
	BuildError:            402, // BuildError indicates generic build error

	// export phase errors: 500-599
	ExportError: 502, // ExportError indicates generic export error

	// rebase phase errors: 600-699
	RebaseError: 602, // RebaseError indicates generic rebase error

	// launch phase errors: 700-799
	LaunchError: 702, // LaunchError indicates generic launch error
}

func (e *LegacyExiter) CodeFor(errType LifecycleExitError) int {
	return codeFor(errType, legacyExitCodes)
}

func codeFor(errType LifecycleExitError, exitCodes map[LifecycleExitError]int) int {
	if code, ok := exitCodes[errType]; ok {
		return code
	}
	return CodeFailed
}
