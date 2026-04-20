package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadyProbe(t *testing.T) {
	tests := []struct {
		name       string
		ready      bool
		wantStatus int
	}{
		{"Ready=true returns 200", true, http.StatusOK},
		{"Ready=false returns 404", false, http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			Ready = tc.ready
			t.Cleanup(func() { Ready = true })

			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			readyProbe(w, r)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestHealthyProbe(t *testing.T) {
	tests := []struct {
		name       string
		healthy    bool
		wantStatus int
	}{
		{"Healty=true returns 200", true, http.StatusOK},
		{"Healty=false returns 404", false, http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			Healty = tc.healthy
			t.Cleanup(func() { Healty = true })

			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			healthyProbe(w, r)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}
