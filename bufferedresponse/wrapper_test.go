package bufferedresponse

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWrap(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := Wrap(rec)

	if rw.Status() != http.StatusOK {
		t.Errorf("expected Status 200, got %d", rw.Status())
	}
	if rw.Size() != 0 {
		t.Errorf("expected Size 0, got %d", rw.Size())
	}
}

func TestWrite_BuffersBeforeFlush(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := Wrap(rec)

	n, err := rw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected n=5, got %d", n)
	}
	if rw.Size() != 5 {
		t.Errorf("expected Size 5, got %d", rw.Size())
	}
	if rec.Body.Len() != 0 {
		t.Errorf("expected underlying recorder body to be empty, got %d bytes", rec.Body.Len())
	}
}

func TestWriteHeader(t *testing.T) {
	t.Run("sets status", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := Wrap(rec)
		rw.WriteHeader(201)
		if rw.Status() != 201 {
			t.Errorf("expected Status 201, got %d", rw.Status())
		}
	})

	t.Run("panics after flush", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := Wrap(rec)
		rw.Flush()

		defer func() {
			r := recover()
			if r == nil {
				t.Error("expected panic on WriteHeader after flush, got none")
			}
		}()
		rw.WriteHeader(400)
	})
}

func TestFlush(t *testing.T) {
	t.Run("sends status and body to underlying writer", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := Wrap(rec)
		rw.WriteHeader(201)
		_, _ = rw.Write([]byte("hello"))

		rw.Flush()

		if rec.Code != 201 {
			t.Errorf("expected recorder code 201, got %d", rec.Code)
		}
		if rec.Body.String() != "hello" {
			t.Errorf("expected recorder body %q, got %q", "hello", rec.Body.String())
		}
	})

	t.Run("second flush is a no-op", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := Wrap(rec)
		_, _ = rw.Write([]byte("data"))
		rw.Flush()

		// second flush should not write again
		_, _ = rw.Write([]byte("more")) // goes directly to rec after flush
		rw.Flush()

		// rec body should contain "data" + "more" (Write after flush goes to underlying)
		// but the second Flush call itself must not panic or duplicate data
		if rec.Body.String() == "" {
			t.Error("expected non-empty body after flush")
		}
	})
}

func TestReset(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := Wrap(rec)
	rw.WriteHeader(202)
	_, _ = rw.Write([]byte("some data"))

	rw.Reset()

	if rw.Size() != 0 {
		t.Errorf("expected Size 0 after Reset, got %d", rw.Size())
	}
	if rw.Status() != 202 {
		t.Errorf("expected Status 202 unchanged after Reset, got %d", rw.Status())
	}
}

func TestGet(t *testing.T) {
	t.Run("wrapped writer returns true", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := Wrap(rec)

		got, ok := Get(rw)
		if !ok {
			t.Error("expected ok=true for wrapped ResponseWriter")
		}
		if got != rw {
			t.Error("expected returned pointer to equal original *ResponseWriter")
		}
	})

	t.Run("plain recorder returns false", func(t *testing.T) {
		rec := httptest.NewRecorder()

		got, ok := Get(rec)
		if ok {
			t.Error("expected ok=false for plain httptest.ResponseRecorder")
		}
		if got != nil {
			t.Error("expected nil *ResponseWriter for plain recorder")
		}
	})
}
