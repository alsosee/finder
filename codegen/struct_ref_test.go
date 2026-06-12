package main

import (
	"testing"
)

func TestStructRef(t *testing.T) {
	tt := []struct {
		ref    string
		prefix string
		escape bool
		want   string
	}{
		{
			ref:    "$ID",
			prefix: "c",
			want:   "c.ID",
		},
		{
			ref:    "$ID/Characters/$name",
			prefix: "character",
			want:   "c.ID + \"/Characters/\" + character.Name",
		},
		{
			ref:    "$$/Characters/$name",
			prefix: "character",
			want:   "c.SourceNoExtention + \"/Characters/\" + character.Name",
		},
		{
			ref:    "$$/Characters/$name",
			prefix: "character",
			escape: true,
			want:   "c.SourceNoExtention + \"/Characters/\" + EscapeFileName(character.Name)",
		},
		{
			ref:    "",
			prefix: "",
			want:   "",
		},
	}

	for _, tc := range tt {
		got := structRef(tc.ref, tc.prefix, tc.escape)
		if got != tc.want {
			t.Fatalf("got %s, expected %s", got, tc.want)
		}
	}
}

func TestFieldTypeCustomIssueArray(t *testing.T) {
	s := Schema{
		Extra: map[string]Content{
			"issue": {},
		},
	}

	got := s.FieldType(Property{
		Name: "issues",
		Type: "array",
		Items: &Property{
			Type: "issue",
		},
	})
	if got != "[]*Issue" {
		t.Fatalf("got %s, expected []*Issue", got)
	}
}

func TestFieldNameAvoidsGeneratedContentFieldCollision(t *testing.T) {
	got := contentFieldName(Property{Name: "awards"})
	if got != "AwardItems" {
		t.Fatalf("got %s, expected AwardItems", got)
	}
}

func TestFieldNameAllowsExtraTypeAwards(t *testing.T) {
	got := fieldName(Property{Name: "awards"})
	if got != "Awards" {
		t.Fatalf("got %s, expected Awards", got)
	}
}

func TestFieldNameUsesNonSemanticAlias(t *testing.T) {
	got := contentFieldName(Property{Name: "awards", Alias: "source_awards"})
	if got != "SourceAwards" {
		t.Fatalf("got %s, expected SourceAwards", got)
	}
}

func TestCustomIssueTypeHasNoConnectionsOrMedia(t *testing.T) {
	schema = &Schema{
		Extra: map[string]Content{
			"issue": {
				Properties: PropertySlice{
					{Name: "number", Type: "string"},
					{Name: "title", Type: "string"},
					{Name: "released", Type: "string"},
				},
			},
		},
	}

	if hasConnections(schema.Extra["issue"]) {
		t.Fatal("issue should not generate connection traversal")
	}
	if hasMedia(schema.Extra["issue"]) {
		t.Fatal("issue should not generate media traversal")
	}
}
