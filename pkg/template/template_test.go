package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jossecurity/joss/pkg/parser"
)

func TestGeneratedProjectsUseCanonicalParsableJoss(t *testing.T) {
	tests := []struct {
		name   string
		create func(string)
	}{
		{name: "web", create: CreateBibleProject},
		{name: "console", create: CreateConsoleProject},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), tt.name)
			tt.create(root)

			err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
				if walkErr != nil || info.IsDir() || filepath.Ext(path) != ".joss" || filepath.Base(path) == "env.joss" {
					return walkErr
				}
				content, readErr := os.ReadFile(path)
				if readErr != nil {
					return readErr
				}
				if strings.Contains(string(content), "function") {
					t.Errorf("%s still generates legacy 'function' syntax", path)
				}
				p := parser.NewParser(parser.NewLexer(string(content)))
				p.ParseProgram()
				if len(p.Errors()) > 0 {
					t.Errorf("%s has parser errors: %v", path, p.Errors())
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
