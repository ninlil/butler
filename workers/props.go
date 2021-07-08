package workers

import "time"

// Name returns the name of the Worker
func (w *Worker) Name() string {
	if w == nil {
		return ""
	}
	return w.name
}

// Started returns starting timestamp of the worker
func (w *Worker) Started() time.Time {
	if w == nil {
		return time.Time{}
	}
	return w.started
}

// Ended returns the ending-timestamp of the worker
func (w *Worker) Ended() time.Time {
	if w == nil {
		return time.Time{}
	}
	return w.ended
}

// IsActive returns if the worker is active or not
func (w *Worker) IsActive() bool {
	if w == nil {
		return false
	}
	return w.state == stateRunning
}
