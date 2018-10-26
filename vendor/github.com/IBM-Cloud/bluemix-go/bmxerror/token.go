package bmxerror

//InvalidTokenError ...
type InvalidTokenError struct {
	Message string
}

//NewInvalidTokenError ...
func NewInvalidTokenError(message string) *InvalidTokenError {
	return &InvalidTokenError{
		Message: message,
	}
}

func (e *InvalidTokenError) Error() string {
	return ("Invalid auth token: ") + e.Message
}
