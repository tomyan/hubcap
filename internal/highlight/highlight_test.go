package highlight

import "testing"

func TestHighlightEmpty(t *testing.T) {
	spans := Highlight("")
	if spans != nil {
		t.Errorf("expected nil for empty input, got %v", spans)
	}
}

func TestHighlightPreservesSource(t *testing.T) {
	cases := []string{
		"document.title",
		"const x = 42",
		`console.log("hello")`,
		"if (x > 0) { return x }",
		"",
	}
	for _, src := range cases {
		spans := Highlight(src)
		text := ""
		for _, s := range spans {
			text += s.Text
		}
		if text != src {
			t.Errorf("Highlight(%q) text = %q, want %q", src, text, src)
		}
	}
}

func TestHighlightKeywords(t *testing.T) {
	spans := Highlight("const x = 1")
	if len(spans) == 0 {
		t.Fatal("expected spans")
	}
	if spans[0].Kind != "keyword" {
		t.Errorf("first span kind = %q, want keyword", spans[0].Kind)
	}
}

func TestHighlightString(t *testing.T) {
	spans := Highlight(`"hello"`)
	found := false
	for _, s := range spans {
		if s.Kind == "string" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected string span in %v", spans)
	}
}

func TestHighlightNumber(t *testing.T) {
	spans := Highlight("42")
	if len(spans) == 0 {
		t.Fatal("expected spans")
	}
	if spans[0].Kind != "number" {
		t.Errorf("span kind = %q, want number", spans[0].Kind)
	}
}

func TestHighlightProperty(t *testing.T) {
	spans := Highlight("obj.prop")
	found := false
	for _, s := range spans {
		if s.Kind == "property" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected property span in %v", spans)
	}
}

func TestHighlightOperator(t *testing.T) {
	spans := Highlight("1 + 2")
	found := false
	for _, s := range spans {
		if s.Kind == "operator" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected operator span in %v", spans)
	}
}

func TestHighlightIncompleteExpression(t *testing.T) {
	// Should not panic on incomplete input.
	cases := []string{"const ", "if (", `"unclosed`, "x."}
	for _, src := range cases {
		spans := Highlight(src)
		text := ""
		for _, s := range spans {
			text += s.Text
		}
		if text != src {
			t.Errorf("Highlight(%q) text = %q, want %q", src, text, src)
		}
	}
}
