package platform

type LifecycleExitError int

const (
	CodeForFailed = 1
)

const (
	FailedDetect             LifecycleExitError = iota // generic detect error
	FailedDetectWithErrors                             // no buildpacks detected
	DetectError                                        // no buildpacks detected and at least one errored
	AnalyzeError                                       // generic analyze error
	RestoreError                                       // generic restore error
	FailedBuildWithErrors                              // buildpack error during /bin/build
	BuildError                                         // generic build error
	ExportError                                        // generic export error
	RebaseError                                        // generic rebase error
	LaunchError                                        // generic launch error
	FailedGenerateWithErrors                           // extension error during /bin/generate
	GenerateError                                      // generic generate error
	ExtendError                                        // generic extend error
)

type Exiter interface {
	CodeFor(errType LifecycleExitError) int
}

// NewExiter configures a new Exiter according to the provided Platform API version.
func NewExiter(_ string) Exiter {
	return &DefaultExiter{}
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

	// generate phase errors: 90-99
	FailedGenerateWithErrors: 91, // FailedGenerateWithErrors indicates extension error during /bin/generate
	GenerateError:            92, // GenerateError indicates generic generate error

	// extend phase errors: 100-109
	ExtendError: 102, // ExtendError indicates generic extend error
}

func (e *DefaultExiter) CodeFor(errType LifecycleExitError) int {
	return codeFor(errType, defaultExitCodes)
}

func codeFor(errType LifecycleExitError, exitCodes map[LifecycleExitError]int) int {
	if code, ok := exitCodes[errType]; ok {
		return code
	}
	return CodeForFailed
}
