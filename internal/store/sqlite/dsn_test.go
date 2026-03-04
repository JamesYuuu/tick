package sqlite

import "testing"

func TestDSNForPath_EscapesSpaces(t *testing.T) {
	path := "/tmp/dir with spaces/todo.db"

	got := dsnForPath(path)
	want := "file:/tmp/dir%20with%20spaces/todo.db"
	if got != want {
		t.Fatalf("dsnForPath(%q) = %q, want %q", path, got, want)
	}
}
