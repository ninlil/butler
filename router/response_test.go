package router

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"
)

func TestGetContentTypeFormat(t *testing.T) {
	tests := []struct {
		name       string
		accept     string
		wantCTF    ctFormat
		wantIndent int
		wantCustom bool
	}{
		{"empty string", "", ctfJSON, 0, false},
		{"application/json", "application/json", ctfJSON, 0, true},
		{"application/xml", "application/xml", ctfXML, 0, true},
		{"text/plain", "text/plain", ctfTEXT, 0, true},
		{"json indent=3", "application/json; indent=3", ctfJSON, 3, true},
		{"json indent=15 clamped", "application/json; indent=15", ctfJSON, 10, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctf, indent, isCustom := getContentTypeFormat(tc.accept, "", "")
			if ctf != tc.wantCTF {
				t.Errorf("ctf = %v, want %v", ctf, tc.wantCTF)
			}
			if indent != tc.wantIndent {
				t.Errorf("indent = %d, want %d", indent, tc.wantIndent)
			}
			if isCustom != tc.wantCustom {
				t.Errorf("isCustom = %v, want %v", isCustom, tc.wantCustom)
			}
		})
	}
}

type testStruct struct {
	XMLName xml.Name `json:"-" xml:"item"`
	Name    string   `json:"name" xml:"name"`
	Value   int      `json:"value" xml:"value"`
}

func TestCreateResponse(t *testing.T) {
	t.Run("JSON marshal of struct", func(t *testing.T) {
		data := testStruct{Name: "foo", Value: 42}
		buf, ct, _, err := createResponse("application/json", data, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(ct, "json") {
			t.Errorf("ct %q does not contain \"json\"", ct)
		}
		if !json.Valid(buf) {
			t.Errorf("buf is not valid JSON: %s", buf)
		}
		var got testStruct
		if err := json.Unmarshal(buf, &got); err != nil {
			t.Fatalf("json.Unmarshal error: %v", err)
		}
		if got.Name != "foo" || got.Value != 42 {
			t.Errorf("unexpected decoded value: %+v", got)
		}
	})

	t.Run("XML marshal of struct", func(t *testing.T) {
		data := testStruct{Name: "bar", Value: 7}
		buf, ct, _, err := createResponse("application/xml", data, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(ct, "xml") {
			t.Errorf("ct %q does not contain \"xml\"", ct)
		}
		var got testStruct
		if err := xml.Unmarshal(buf, &got); err != nil {
			t.Fatalf("xml.Unmarshal error: %v", err)
		}
		if got.Name != "bar" || got.Value != 7 {
			t.Errorf("unexpected decoded value: %+v", got)
		}
	})

	t.Run("text/plain with string", func(t *testing.T) {
		buf, ct, _, err := createResponse("text/plain", "hello", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(ct, "text") {
			t.Errorf("ct %q does not contain \"text\"", ct)
		}
		if !bytes.Equal(buf, []byte("hello")) {
			t.Errorf("buf = %q, want %q", buf, "hello")
		}
	})

	t.Run("text/plain with []string", func(t *testing.T) {
		buf, _, _, err := createResponse("text/plain", []string{"line1", "line2", "line3"}, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "line1\nline2\nline3"
		if string(buf) != want {
			t.Errorf("buf = %q, want %q", buf, want)
		}
	})

	t.Run("raw []byte returned as-is", func(t *testing.T) {
		raw := []byte{0x01, 0x02, 0x03}
		buf, _, _, err := createResponse("application/json", raw, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(buf, raw) {
			t.Errorf("buf = %v, want %v", buf, raw)
		}
	})

	t.Run("nil data with no custom Accept", func(t *testing.T) {
		buf, _, _, err := createResponse("", nil, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf != nil {
			t.Errorf("expected nil buf, got %v", buf)
		}
	})
}
