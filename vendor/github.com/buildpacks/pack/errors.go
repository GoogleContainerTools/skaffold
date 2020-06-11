package pack

// ExperimentError denotes that an experimental feature was trying to be used without experimental features enabled.
type ExperimentError struct {
	msg string
}

func NewExperimentError(msg string) ExperimentError {
	return ExperimentError{msg}
}

func (ee ExperimentError) Error() string {
	return ee.msg
}

// SoftError is an error that is not intended to be displayed.
type SoftError struct{}

func NewSoftError() SoftError {
	return SoftError{}
}

func (se SoftError) Error() string {
	return ""
}
