package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
)

// TestFromGitDiff_SubdirAndUnreadable covers the two failure modes seen in the
// field: walle run from a subdirectory of the repo (git paths are root-relative)
// and a file that exists in git status but cannot be read.
func TestFromGitDiff_SubdirAndUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read chmod 000 files")
	}

	root := t.TempDir()
	if _, err := git.PlainInit(root, false); err != nil {
		t.Fatal(err)
	}

	sub := filepath.Join(root, "backend")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(root, "context", "confirm.js")
	if err := os.MkdirAll(filepath.Dir(good), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(good, []byte("// hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(root, "secret.js")
	if err := os.WriteFile(bad, []byte("x"), 0o000); err != nil {
		t.Fatal(err)
	}

	collect, err := FromGitDiff(sub)
	if err != nil {
		t.Fatal(err)
	}
	files, warnings, err := collect()
	if err != nil {
		t.Fatalf("collect must not fail on a single unreadable file: %v", err)
	}
	if len(files) != 1 || files[0].Path != good {
		t.Fatalf("want [%s], got %+v", good, files)
	}
	if len(warnings) != 1 || warnings[0].Path != bad {
		t.Fatalf("want one warning for %s, got %+v", bad, warnings)
	}
}
