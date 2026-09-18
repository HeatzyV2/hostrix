package containers

import "testing"

func TestValidateContainerName(t *testing.T) {
	if err := ValidateContainerName("hx-abc123"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateContainerName("../etc"); err == nil {
		t.Fatal("expected invalid name")
	}
}

func TestSanitizeContainerPath(t *testing.T) {
	got, err := SanitizeContainerPath("../../etc/passwd")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/etc/passwd" {
		// Clean collapses .. against root to /etc/passwd which is INSIDE container.
		// That is correct for container FS; host escape is prevented by Incus file API scope.
		t.Fatalf("got %q", got)
	}
	if _, err := SanitizeContainerPath(""); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateBackupName(t *testing.T) {
	if err := ValidateBackupName("550e8400-e29b-41d4-a716-446655440000"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateBackupName("../bad"); err == nil {
		t.Fatal("expected invalid backup name")
	}
}

func TestValidateCreateRequest(t *testing.T) {
	err := ValidateCreateRequest(CreateRequest{
		Name: "hx-test", Image: "ubuntu/24.04", MemoryMB: 512, CPULimit: 100, DiskMB: 2048,
	})
	if err != nil {
		t.Fatal(err)
	}
}
