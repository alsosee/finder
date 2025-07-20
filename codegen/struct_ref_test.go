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
