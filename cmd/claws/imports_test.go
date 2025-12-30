package main

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/clawscli/claws/internal/genimports"
)

func TestAllRegisterPackagesImported(t *testing.T) {
	projectRoot, err := genimports.GetProjectRoot()
	if err != nil {
		t.Fatalf("failed to get project root: %v", err)
	}

	registerPackages, err := genimports.FindRegisterPackages(projectRoot)
	if err != nil {
		t.Fatalf("failed to find register packages: %v", err)
	}

	importedPackages, err := parseImportedPackages("imports_custom.go")
	if err != nil {
		t.Fatalf("failed to parse imports_custom.go: %v", err)
	}

	var missing []string
	for _, pkg := range registerPackages {
		if !importedPackages[pkg] {
			missing = append(missing, pkg)
		}
	}

	if len(missing) > 0 {
		t.Errorf("The following packages have register.go but are not imported in imports_custom.go:\n")
		for _, pkg := range missing {
			t.Errorf("  _ %q", pkg)
		}
		t.Errorf("\nRun 'task gen-imports' to regenerate imports_custom.go")
	}
}

func parseImportedPackages(filename string) (map[string]bool, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	imported := make(map[string]bool)
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		imported[path] = true
	}

	return imported, nil
}

func TestImportsCustomFileExists(t *testing.T) {
	if _, err := os.Stat("imports_custom.go"); os.IsNotExist(err) {
		t.Fatal("imports_custom.go does not exist. Run 'task gen-imports' to generate it.")
	}
}

func TestNoBlankImportsInMainGo(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "main.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("failed to parse main.go: %v", err)
	}

	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if strings.Contains(path, "clawscli/claws/custom/") {
			if imp.Name != nil && imp.Name.Name == "_" {
				t.Errorf("main.go should not contain blank imports from custom/. Found: %s", path)
				t.Errorf("Move this import to imports_custom.go")
			}
		}
	}
}

func TestImportsCustomHasGeneratedHeader(t *testing.T) {
	content, err := os.ReadFile("imports_custom.go")
	if err != nil {
		t.Fatalf("failed to read imports_custom.go: %v", err)
	}

	if !strings.Contains(string(content), "Code generated") {
		t.Error("imports_custom.go should have a 'Code generated' header comment")
	}
}

func TestAllImportsAreBlankImports(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "imports_custom.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("failed to parse imports_custom.go: %v", err)
	}

	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if !strings.Contains(path, "clawscli/claws/custom/") {
			continue
		}

		if imp.Name == nil || imp.Name.Name != "_" {
			t.Errorf("import %q should be a blank import (use _)", path)
		}
	}
}
