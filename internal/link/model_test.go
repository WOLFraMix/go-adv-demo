package link_test

import (
	"go-adv-demo/internal/link"
	"testing"
)

func TestNewLinkGeneratesLowercaseHash(t *testing.T) {
	l := link.NewLink("https://example.com")
	if l.Url != "https://example.com" {
		t.Fatalf("expected url to be kept, got %q", l.Url)
	}
	if len(l.Hash) != 10 {
		t.Fatalf("expected 10-char hash, got %q", l.Hash)
	}
	for _, r := range l.Hash {
		if r < 'a' || r > 'z' {
			t.Fatalf("hash %q contains non a-z rune %q", l.Hash, r)
		}
	}
}
