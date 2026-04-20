package router

import (
	"errors"
	"strings"
	"testing"
)

func TestFieldError_Error(t *testing.T) {
	fe := &FieldError{Name: "myfield", Message: "something went wrong"}
	got := fe.Error()
	if !strings.Contains(got, "myfield") {
		t.Errorf("Error() = %q, want it to contain %q", got, "myfield")
	}
	if !strings.Contains(got, "something went wrong") {
		t.Errorf("Error() = %q, want it to contain %q", got, "something went wrong")
	}
}

func TestNewFieldError_NilErr(t *testing.T) {
	fe := newFieldError(nil, "field1", 42, "bad value")
	if fe == nil {
		t.Fatal("expected non-nil *FieldError")
	}
	if fe.Name != "field1" {
		t.Errorf("Name = %q, want %q", fe.Name, "field1")
	}
	if fe.Value != 42 {
		t.Errorf("Value = %v, want %v", fe.Value, 42)
	}
	if fe.Message != "bad value" {
		t.Errorf("Message = %q, want %q", fe.Message, "bad value")
	}
}

func TestNewFieldError_ExistingFieldError(t *testing.T) {
	existing := &FieldError{Name: "orig", Message: "original error", Value: "origval"}
	// Capture err.Error() before the call; the implementation mutates the same pointer
	// (sets Name, clears Message) before calling err.Error(), so the resulting message
	// will be "field-error on '<new_name>': ".
	wantMessage := (&FieldError{Name: "new_name", Message: ""}).Error()

	fe := newFieldError(existing, "new_name", "new_val", "")

	// Should return the same *FieldError (not a fresh one)
	if fe != existing {
		t.Errorf("expected the existing *FieldError to be returned, got a different pointer")
	}
	if fe.Name != "new_name" {
		t.Errorf("Name = %q, want %q", fe.Name, "new_name")
	}
	if fe.Value != "new_val" {
		t.Errorf("Value = %v, want %v", fe.Value, "new_val")
	}
	if fe.Message != wantMessage {
		t.Errorf("Message = %q, want %q", fe.Message, wantMessage)
	}
}

func TestNewFieldError_PlainError(t *testing.T) {
	plain := errors.New("plain error message")
	fe := newFieldError(plain, "pfield", "pval", "")

	if fe == nil {
		t.Fatal("expected non-nil *FieldError")
	}
	if fe.Name != "pfield" {
		t.Errorf("Name = %q, want %q", fe.Name, "pfield")
	}
	if fe.Value != "pval" {
		t.Errorf("Value = %v, want %v", fe.Value, "pval")
	}
	if fe.Message != plain.Error() {
		t.Errorf("Message = %q, want %q", fe.Message, plain.Error())
	}
}
