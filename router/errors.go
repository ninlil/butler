package router

import (
	"errors"
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
	ErrInvalidMatch         = fmt.Errorf("invalid match")
	ErrRouterDuplicateName  = fmt.Errorf("duplicate router name")
)

// FieldError is the error-message returned when a parameter (query och path) is invalid
type FieldError struct {
	Name    string      `json:"name"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// Error to act as an 'error'-type
func (fe *FieldError) Error() string {
	return fmt.Sprintf("field-error on '%s': %s", fe.Name, fe.Message)
}

func newFieldError(err error, name string, v interface{}, msg string) *FieldError {
	var fe *FieldError

	if err != nil {
		var fe2 *FieldError
		if errors.As(err, &fe2) {
			fe = fe2
		}
	}

	if fe == nil {
		fe = new(FieldError)
	}

	fe.Name = name
	fe.Value = v
	fe.Message = msg
	if fe.Message == "" && err != nil {
		fe.Message = err.Error()
	}

	return fe
}
