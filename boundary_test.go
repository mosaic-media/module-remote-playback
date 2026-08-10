package remoteplayback_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestModuleImportsOnlyPublishedContracts is the module boundary made
// executable: this module must use only the published SDK and the standard
// library. It is a separate Go module, so Go itself already rejects a
// Platform-internal import; this parse keeps the intent explicit and catches a
// third-party dependency creeping in too (sdk#1, platform#12).
//
// It matters more here than for a source module. A consumer is the one that
// wants to reach the network, hold a credential and eventually touch the disk,
// so it is the one most likely to grow a dependency it should not have — and it
// is exactly the kind of module a third party will write.
func TestModuleImportsOnlyPublishedContracts(t *testing.T) {
	const (
		sdkPrefix      = "github.com/mosaic-media/sdk/"
		platformPrefix = "github.com/mosaic-media/platform/"
	)

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}

	fset := token.NewFileSet()
	checked := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		checked++

		file, err := parser.ParseFile(fset, filepath.Join(".", name), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imp := range file.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("unquote import in %s: %v", name, err)
			}
			switch {
			// Standard-library imports have no dot in their first segment.
			case !strings.Contains(strings.SplitN(path, "/", 2)[0], "."):
			case strings.HasPrefix(path, sdkPrefix):
				// The published SDK — the primary contract a module builds against.
			case strings.HasPrefix(path, platformPrefix):
				t.Errorf("%s imports private Platform package %q; a module may import only the SDK", name, path)
			default:
				t.Errorf("%s imports third-party package %q; this module may use only the SDK and the standard library", name, path)
			}
		}
	}

	if checked == 0 {
		t.Fatal("no non-test source files were checked; the boundary test is not looking at anything")
	}
}
