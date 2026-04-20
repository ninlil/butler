package workers

import (
	"context"
	"testing"
)

func resetDriver() {
	driver = new(driverType)
}

func TestNew_DuplicatePanics(t *testing.T) {
	resetDriver()
	t.Cleanup(resetDriver)

	fn := func(ctx context.Context) {}
	New("dup-alpha", fn)

	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic on duplicate worker name, got none")
		}
	}()
	New("dup-alpha", fn)
}

func TestGet_FromWorkerContext(t *testing.T) {
	resetDriver()
	t.Cleanup(resetDriver)

	fn := func(ctx context.Context) {}
	New("get-beta", fn)

	w := driver.list["get-beta"]
	if w == nil {
		t.Fatal("worker 'get-beta' not found in driver")
	}

	got := Get(w.ctx)
	if got == nil {
		t.Fatal("Get returned nil for worker context")
	}
	if got.Name() != "get-beta" {
		t.Errorf("Name() = %q, want %q", got.Name(), "get-beta")
	}
}

func TestGet_BackgroundContext(t *testing.T) {
	got := Get(context.Background())
	if got != nil {
		t.Errorf("Get(Background) = %v, want nil", got)
	}
}

func TestNilWorker_Name(t *testing.T) {
	var w *Worker
	if got := w.Name(); got != "" {
		t.Errorf("(*Worker)(nil).Name() = %q, want \"\"", got)
	}
}

func TestNilWorker_Started(t *testing.T) {
	var w *Worker
	if got := w.Started(); !got.IsZero() {
		t.Errorf("(*Worker)(nil).Started() = %v, want zero time", got)
	}
}

func TestNilWorker_Ended(t *testing.T) {
	var w *Worker
	if got := w.Ended(); !got.IsZero() {
		t.Errorf("(*Worker)(nil).Ended() = %v, want zero time", got)
	}
}

func TestNilWorker_IsActive(t *testing.T) {
	var w *Worker
	if got := w.IsActive(); got {
		t.Error("(*Worker)(nil).IsActive() = true, want false")
	}
}
