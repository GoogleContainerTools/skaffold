package bmxerror

//Error interface
type Error interface {
	Error() string
	Code() string
	Description() string
}

//RequestFailure interface
type RequestFailure interface {
	Error
	// The status code of the HTTP response.
	StatusCode() int
}

//New creates a new Error object
func New(code, description string) Error {
	return newGenericError(code, description)
}

//NewRequestFailure creates a new Error object wrapping the server error
func NewRequestFailure(code, description string, statusCode int) Error {
	return newRequestError(code, description, statusCode)
}
