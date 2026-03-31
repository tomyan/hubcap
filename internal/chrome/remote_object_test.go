package chrome

import "testing"

func TestFormatPrimitiveString(t *testing.T) {
	obj := &RemoteObject{Type: "string", Value: "hello"}
	if got := FormatRemoteObject(obj); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestFormatPrimitiveNumber(t *testing.T) {
	obj := &RemoteObject{Type: "number", Value: float64(42)}
	if got := FormatRemoteObject(obj); got != "42" {
		t.Errorf("got %q, want %q", got, "42")
	}
}

func TestFormatBoolean(t *testing.T) {
	obj := &RemoteObject{Type: "boolean", Value: true}
	if got := FormatRemoteObject(obj); got != "true" {
		t.Errorf("got %q, want %q", got, "true")
	}
}

func TestFormatUndefined(t *testing.T) {
	obj := &RemoteObject{Type: "undefined"}
	if got := FormatRemoteObject(obj); got != "undefined" {
		t.Errorf("got %q, want %q", got, "undefined")
	}
}

func TestFormatNull(t *testing.T) {
	obj := &RemoteObject{Type: "object", Subtype: "null"}
	if got := FormatRemoteObject(obj); got != "null" {
		t.Errorf("got %q, want %q", got, "null")
	}
}

func TestFormatObjectWithPreview(t *testing.T) {
	obj := &RemoteObject{
		Type:      "object",
		ClassName: "Object",
		ObjectID:  "1234",
		Preview: &ObjectPreview{
			Type: "object",
			Properties: []PropertyPreview{
				{Name: "name", Type: "string", Value: "foo"},
				{Name: "count", Type: "number", Value: "3"},
			},
		},
	}
	got := FormatRemoteObject(obj)
	if got != `{name: foo, count: 3}` {
		t.Errorf("got %q", got)
	}
}

func TestFormatArrayPreview(t *testing.T) {
	obj := &RemoteObject{
		Type:      "object",
		Subtype:   "array",
		ClassName: "Array",
		ObjectID:  "5678",
		Preview: &ObjectPreview{
			Type:    "object",
			Subtype: "array",
			Properties: []PropertyPreview{
				{Name: "0", Type: "number", Value: "1"},
				{Name: "1", Type: "number", Value: "2"},
				{Name: "2", Type: "number", Value: "3"},
				{Name: "length", Type: "number", Value: "3"},
			},
		},
	}
	got := FormatRemoteObject(obj)
	if got != "[1, 2, 3]" {
		t.Errorf("got %q, want %q", got, "[1, 2, 3]")
	}
}

func TestFormatOverflow(t *testing.T) {
	obj := &RemoteObject{
		Type:    "object",
		Preview: &ObjectPreview{
			Type: "object",
			Properties: []PropertyPreview{
				{Name: "a", Type: "number", Value: "1"},
			},
			Overflow: true,
		},
	}
	got := FormatRemoteObject(obj)
	if got != "{a: 1, …}" {
		t.Errorf("got %q, want %q", got, "{a: 1, …}")
	}
}

func TestFormatUnserializable(t *testing.T) {
	obj := &RemoteObject{Type: "number", UnserializableValue: "Infinity"}
	if got := FormatRemoteObject(obj); got != "Infinity" {
		t.Errorf("got %q, want %q", got, "Infinity")
	}
}
