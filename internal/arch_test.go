package internal_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const (
	modulePrefix = "github.com/soundcheck/soundcheck"
	corePrefix   = modulePrefix + "/internal/core"
)

// Banned: non-stdlib deps allowed only in internal/edge/** + internal/transport/**.
var bannedCoreImports = []string{
	"github.com/yuin/gopher-lua",
	"github.com/creack/pty",
	"golang.org/x/crypto/ssh",
	"golang.org/x/sys",
}

func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate go.mod from %s", dir)
		}
		dir = parent
	}
}

// isAllowedImport classifies an import path for internal/core/**.
func isAllowedImport(imp string) bool {
	for _, b := range bannedCoreImports {
		if imp == b || strings.HasPrefix(imp, b+"/") {
			return false
		}
	}
	if imp == corePrefix || strings.HasPrefix(imp, corePrefix+"/") {
		return true
	}
	// ponytail: first-segment no-dot check rejects all remaining non-stdlib
	// packages in one pass; add a package here when a new stdlib one appears.
	first := imp
	if i := strings.IndexByte(imp, '/'); i >= 0 {
		first = imp[:i]
	}
	return !strings.Contains(first, ".")
}

func TestCorePackageImportsStdlibOnly(t *testing.T) {
	moduleRoot := findModuleRoot(t)
	coreDir := filepath.Join(moduleRoot, "internal", "core")

	if _, err := os.Stat(coreDir); err != nil {
		t.Fatalf("internal/core directory does not exist: %v", err)
	}

	fset := token.NewFileSet()
	checkedFiles := 0

	err := filepath.WalkDir(coreDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}

		checkedFiles++
		fileAst, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			t.Errorf("failed to parse %s: %v", path, parseErr)
			return nil
		}

		for _, imp := range fileAst.Imports {
			impPath, unquoteErr := strconv.Unquote(imp.Path.Value)
			if unquoteErr != nil {
				t.Errorf("%s: invalid import string literal %s: %v", path, imp.Path.Value, unquoteErr)
				continue
			}

			if !isAllowedImport(impPath) {
				relPath, _ := filepath.Rel(moduleRoot, path)
				t.Errorf("architectural violation: %s imports %q; core packages must only import standard library or intra-core packages", relPath, impPath)
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("error walking internal/core: %v", err)
	}

	if checkedFiles == 0 {
		t.Fatalf("no .go files found in %s", coreDir)
	}
}
