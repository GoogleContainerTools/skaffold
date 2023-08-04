package buildpack

type ErrorType string

const ErrTypeBuildpack ErrorType = "ERR_BUILDPACK"
const ErrTypeFailedDetection ErrorType = "ERR_FAILED_DETECTION"

type Error struct {
	RootError error
	Type      ErrorType
}

func (le *Error) Error() string {
	if le.Cause() != nil {
		return le.Cause().Error()
	}
	return string(le.Type)
}

func (le *Error) Cause() error {
	return le.RootError
}

func NewError(cause error, errType ErrorType) *Error {
	return &Error{RootError: cause, Type: errType}
}
