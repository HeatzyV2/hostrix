package containers

import "testing"

func TestSanitizeContainerPathAllowRoot(t *testing.T) {
	got, err := SanitizeContainerPathAllowRoot("/")
	if err != nil || got != "/" {
		t.Fatalf("got %q err %v", got, err)
	}
	got, err = SanitizeContainerPathAllowRoot("")
	if err != nil || got != "/" {
		t.Fatalf("empty -> / got %q err %v", got, err)
	}
	got, err = SanitizeContainerPathAllowRoot("/home/../tmp")
	if err != nil || got != "/tmp" {
		t.Fatalf("got %q err %v", got, err)
	}
}
