package bmxerror

import "fmt"

type genericError struct {
	code        string
	description string
}

func newGenericError(code, description string) *genericError {
	return &genericError{code, description}
}

func (g genericError) Error() string {
	return fmt.Sprintf("%s: %s", g.code, g.description)
}

func (g genericError) String() string {
	return g.Error()
}

func (g genericError) Code() string {
	return g.code
}

func (g genericError) Description() string {
	return g.description
}

type requestError struct {
	genericError
	statusCode int
}

func newRequestError(code, description string, statusCode int) *requestError {
	return &requestError{
		genericError: genericError{
			code:        code,
			description: description,
		},
		statusCode: statusCode,
	}
}

func (r requestError) Error() string {
	return fmt.Sprintf("Request failed with status code: %d, %s: %s", r.statusCode, r.code, r.description)
}

func (r requestError) Code() string {
	return r.code
}
func (r requestError) Description() string {
	return r.description
}
func (r requestError) StatusCode() int {
	return r.statusCode
}
