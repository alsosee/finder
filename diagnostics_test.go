package main

import "testing"

func TestFormatDiagnosticsSummaryGroupsUnknownFieldsByTypeAndField(t *testing.T) {
	diagnostics := []Diagnostic{
		{
			File:   "Shows/Example.yml",
			Line:   6,
			Column: 5,
			Type:   "character",
			Field:  "description",
		},
		{
			File:   "Movies/Example.yml",
			Line:   4,
			Column: 3,
			Type:   "content",
			Field:  "born",
		},
		{
			File:   "Shows/Example.yml",
			Line:   8,
			Column: 5,
			Type:   "character",
			Field:  "description",
		},
	}

	got := FormatDiagnosticsSummary(diagnostics)
	want := `Unknown fields summary (3 warnings):
  character:
    description (2): Shows/Example.yml:6:5, Shows/Example.yml:8:5
  content:
    born (1): Movies/Example.yml:4:3`
	if got != want {
		t.Fatalf("summary mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestDiagnosticLogAnnotationIncludesLocationInMessage(t *testing.T) {
	diagnostic := Diagnostic{
		File:    "Movies/Example.yml",
		Line:    4,
		Column:  3,
		Message: `Unknown field "born" at born`,
	}

	if got := diagnostic.AnnotationMessage(); got != `Movies/Example.yml:4:3: Unknown field "born" at born` {
		t.Fatalf("got message %q", got)
	}
}
