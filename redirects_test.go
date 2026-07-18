package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRedirects(t *testing.T) {
	rules, err := ParseRedirects(`
# comment
/Series /Shows 301
/Series/* /Shows/:splat 308
`, "test")
	if err != nil {
		t.Fatalf("ParseRedirects() error = %v", err)
	}

	if len(rules) != 2 {
		t.Fatalf("ParseRedirects() len = %d, want 2", len(rules))
	}
	if rules[0] != (RedirectRule{From: "/Series", To: "/Shows", Status: 301}) {
		t.Fatalf("rules[0] = %#v", rules[0])
	}
	if rules[1] != (RedirectRule{From: "/Series/*", To: "/Shows/:splat", Status: 308}) {
		t.Fatalf("rules[1] = %#v", rules[1])
	}
}

func TestParseRedirectsRejectsUnsupportedWildcard(t *testing.T) {
	_, err := ParseRedirects("/Series/*/bad /Shows/:splat 301", "test")
	if err == nil {
		t.Fatal("ParseRedirects() error = nil, want error")
	}
}

func TestParseRedirectsFileMissingIsEmpty(t *testing.T) {
	rules, err := ParseRedirectsFile(filepath.Join(t.TempDir(), "_redirects"))
	if err != nil {
		t.Fatalf("ParseRedirectsFile() error = %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("ParseRedirectsFile() len = %d, want 0", len(rules))
	}
}

func TestWriteWorkerRedirectsModule(t *testing.T) {
	output := filepath.Join(t.TempDir(), "redirects.generated.js")
	rules := []RedirectRule{{From: "/Series/*", To: "/Shows/:splat", Status: 301}}

	if err := WriteWorkerRedirectsModule(output, rules); err != nil {
		t.Fatalf("WriteWorkerRedirectsModule() error = %v", err)
	}

	b, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(b)
	for _, want := range []string{
		"export const REDIRECTS = [",
		`"from": "/Series/*"`,
		"export const HAS_REDIRECTS = true;",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("generated module missing %q:\n%s", want, got)
		}
	}
}

func TestWriteWorkerRedirectsModuleEmpty(t *testing.T) {
	output := filepath.Join(t.TempDir(), "redirects.generated.js")

	if err := WriteWorkerRedirectsModule(output, nil); err != nil {
		t.Fatalf("WriteWorkerRedirectsModule() error = %v", err)
	}

	b, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(b)
	if !strings.Contains(got, "export const REDIRECTS = [];") {
		t.Fatalf("generated module missing empty table:\n%s", got)
	}
	if !strings.Contains(got, "export const HAS_REDIRECTS = false;") {
		t.Fatalf("generated module missing HAS_REDIRECTS false:\n%s", got)
	}
}
