package router

import (
	"fmt"
)

const (
	errMsgBelowMin    = "value is below minimun %v"
	errMsgAboveMax    = "value is above maximum %v"
	errMsgRequired    = "value is required"
	errMsgUnknownType = "unknown field-type %s"
)

// Our errors
var (
	ErrRouterAlreadyRunning = fmt.Errorf("router is already running")
)

// FieldError is the error-message returned when a parameter (query och path) is invalid
type FieldError struct {
	Name    string      `json:"name"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// Error to act as an 'error'-type
func (fe FieldError) Error() string {
	return fmt.Sprintf("field-error on %s '%v': %s", fe.Name, fe.Value, fe.Message)
}

func newFieldError(name string, v interface{}, msg string) FieldError {
	return FieldError{
		Name:    name,
		Message: msg,
		Value:   v,
	}
}
