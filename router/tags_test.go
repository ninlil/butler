package router

import (
	"reflect"
	"testing"
	"time"
)

// --- parseTag tests ---

func TestParseTag_From(t *testing.T) {
	tests := []struct {
		name     string
		tag      reflect.StructTag
		wantFrom fromSource
	}{
		{"path explicit", `from:"path"`, fromPath},
		{"query", `from:"query"`, fromQuery},
		{"header", `from:"header"`, fromHeader},
		{"body", `from:"body"`, fromBody},
		{"cookie", `from:"cookie"`, fromCookie},
		{"absent defaults to path", ``, fromPath},
		{"unknown defaults to path", `from:"unknown"`, fromPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTag(tt.tag)
			if got.From != tt.wantFrom {
				t.Errorf("From = %d, want %d", got.From, tt.wantFrom)
			}
		})
	}
}

func TestParseTag_Required(t *testing.T) {
	tests := []struct {
		name    string
		tag     reflect.StructTag
		wantReq bool
	}{
		{"present with no value", `required:""`, true},
		{"present with true", `required:"true"`, true},
		{"present with false", `required:"false"`, false},
		{"absent", ``, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTag(tt.tag)
			if got.Required != tt.wantReq {
				t.Errorf("Required = %v, want %v", got.Required, tt.wantReq)
			}
		})
	}
}

func TestParseTag_MinMaxDefaultRegex(t *testing.T) {
	tag := reflect.StructTag(`min:"1" max:"100" default:"42" regex:"^[0-9]+"`)
	got := parseTag(tag)
	if !got.HasMin || got.Min != "1" {
		t.Errorf("HasMin/Min: got %v/%q, want true/\"1\"", got.HasMin, got.Min)
	}
	if !got.HasMax || got.Max != "100" {
		t.Errorf("HasMax/Max: got %v/%q, want true/\"100\"", got.HasMax, got.Max)
	}
	if !got.HasDefault || got.Default != "42" {
		t.Errorf("HasDefault/Default: got %v/%q, want true/\"42\"", got.HasDefault, got.Default)
	}
	if !got.hasRegex || got.Regex != "^[0-9]+" {
		t.Errorf("hasRegex/Regex: got %v/%q, want true/\"^[0-9]+\"", got.hasRegex, got.Regex)
	}
}

// --- tagInfo.int tests ---

func newIntField() reflect.Value {
	var i int
	return reflect.ValueOf(&i).Elem()
}

func TestTagInfo_Int(t *testing.T) { //nolint:dupl
	tests := []struct {
		name    string
		tag     tagInfo
		input   string
		force   bool
		wantErr bool
		wantVal int64
	}{
		{"valid", tagInfo{HasMin: true, Min: "1", HasMax: true, Max: "10"}, "5", false, false, 5},
		{"below min", tagInfo{HasMin: true, Min: "1"}, "0", false, true, 0},
		{"above max", tagInfo{HasMax: true, Max: "10"}, "11", false, true, 0},
		{"force bypasses min", tagInfo{HasMin: true, Min: "5"}, "1", true, false, 1},
		{"force bypasses max", tagInfo{HasMax: true, Max: "5"}, "10", true, false, 10},
		{"invalid value", tagInfo{}, "abc", false, true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newIntField()
			err := tt.tag.int(f, tt.input, tt.force)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && f.Int() != tt.wantVal {
				t.Errorf("value = %d, want %d", f.Int(), tt.wantVal)
			}
		})
	}
}

// --- tagInfo.float tests ---

func newFloatField() reflect.Value {
	var f float64
	return reflect.ValueOf(&f).Elem()
}

func TestTagInfo_Float(t *testing.T) { //nolint:dupl
	tests := []struct {
		name    string
		tag     tagInfo
		input   string
		force   bool
		wantErr bool
		wantVal float64
	}{
		{"valid", tagInfo{HasMin: true, Min: "1.0", HasMax: true, Max: "10.0"}, "5.0", false, false, 5.0},
		{"below min", tagInfo{HasMin: true, Min: "1.0"}, "0.5", false, true, 0},
		{"above max", tagInfo{HasMax: true, Max: "10.0"}, "11.0", false, true, 0},
		{"force bypasses min", tagInfo{HasMin: true, Min: "5.0"}, "1.0", true, false, 1.0},
		{"force bypasses max", tagInfo{HasMax: true, Max: "5.0"}, "10.0", true, false, 10.0},
		{"invalid value", tagInfo{}, "abc", false, true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFloatField()
			err := tt.tag.float(f, tt.input, tt.force)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && f.Float() != tt.wantVal {
				t.Errorf("value = %g, want %g", f.Float(), tt.wantVal)
			}
		})
	}
}

// --- tagInfo.duration tests ---

func newDurationField() reflect.Value {
	var d time.Duration
	return reflect.ValueOf(&d).Elem()
}

func TestTagInfo_Duration(t *testing.T) {
	tests := []struct {
		name    string
		tag     tagInfo
		input   string
		force   bool
		wantErr bool
		wantVal time.Duration
	}{
		{"valid 1s", tagInfo{}, "1s", false, false, time.Second},
		{"below min (30s < 1m)", tagInfo{HasMin: true, Min: "1m"}, "30s", false, true, 0},
		{"above max", tagInfo{HasMax: true, Max: "10s"}, "20s", false, true, 0},
		{"force bypasses min", tagInfo{HasMin: true, Min: "1m"}, "30s", true, false, 30 * time.Second},
		{"force bypasses max", tagInfo{HasMax: true, Max: "10s"}, "20s", true, false, 20 * time.Second},
		{"invalid value", tagInfo{}, "xyz", false, true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newDurationField()
			err := tt.tag.duration(f, tt.input, tt.force)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				got := time.Duration(f.Int())
				if got != tt.wantVal {
					t.Errorf("value = %v, want %v", got, tt.wantVal)
				}
			}
		})
	}
}

// --- tagInfo.bool tests ---

func newBoolField() reflect.Value {
	var b bool
	return reflect.ValueOf(&b).Elem()
}

func TestTagInfo_Bool(t *testing.T) {
	tag := tagInfo{}
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantVal bool
	}{
		{"true", "true", false, true},
		{"false", "false", false, false},
		{"1", "1", false, true},
		{"0", "0", false, false},
		{"invalid string", "yes", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newBoolField()
			err := tag.bool(f, tt.input, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && f.Bool() != tt.wantVal {
				t.Errorf("value = %v, want %v", f.Bool(), tt.wantVal)
			}
		})
	}
}

// --- tagInfo.string tests ---

func newStringField() reflect.Value {
	var s string
	return reflect.ValueOf(&s).Elem()
}

func TestTagInfo_String(t *testing.T) {
	tests := []struct {
		name    string
		tag     tagInfo
		input   string
		force   bool
		wantErr bool
		wantVal string
	}{
		{"valid", tagInfo{HasMin: true, Min: "2", HasMax: true, Max: "10"}, "hello", false, false, "hello"},
		{"below min length", tagInfo{HasMin: true, Min: "5"}, "hi", false, true, ""},
		{"above max length", tagInfo{HasMax: true, Max: "3"}, "hello", false, true, ""},
		{"force bypasses min", tagInfo{HasMin: true, Min: "5"}, "hi", true, false, "hi"},
		{"force bypasses max", tagInfo{HasMax: true, Max: "3"}, "hello", true, false, "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newStringField()
			err := tt.tag.string(f, tt.input, tt.force)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && f.String() != tt.wantVal {
				t.Errorf("value = %q, want %q", f.String(), tt.wantVal)
			}
		})
	}
}
