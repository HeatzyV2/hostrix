package templates

import "testing"

func TestSubstitute(t *testing.T) {
	got := Substitute("java -Xms{{RAM}}M -Xmx{{RAM}}M -jar {{SERVER_JAR}} :{{SERVER_PORT}}", Vars{
		RAM:        2048,
		CPU:        100,
		ServerPort: 25565,
		Extra:      map[string]string{"SERVER_JAR": "server.jar"},
	})
	want := "java -Xms2048M -Xmx2048M -jar server.jar :25565"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSubstituteLeavesUnknown(t *testing.T) {
	got := Substitute("echo {{UNKNOWN}} {{RAM}}", Vars{RAM: 512})
	want := "echo {{UNKNOWN}} 512"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestApplyStartup(t *testing.T) {
	def := &Definition{
		Startup:     StartupSpec{Command: "php -S 0.0.0.0:{{SERVER_PORT}} -t {{DOCROOT}}"},
		Ports:       []int{8080},
		Environment: map[string]string{"DOCROOT": "public"},
	}
	got := ApplyStartup(def, 1024, 100, 0)
	want := "php -S 0.0.0.0:8080 -t public"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
