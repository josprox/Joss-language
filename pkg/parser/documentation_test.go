package parser

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var jossFence = regexp.MustCompile("(?s)```joss\\s*\\n(.*?)```")

func TestPublishedJossExamplesParse(t *testing.T) {
	_, currentFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	paths := []string{filepath.Join(repoRoot, "README.md")}

	err := filepath.Walk(filepath.Join(repoRoot, "docs"), func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !info.IsDir() && strings.EqualFold(filepath.Ext(path), ".md") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		for index, match := range jossFence.FindAllStringSubmatch(string(data), -1) {
			parser := NewParser(NewLexer(match[1]))
			parser.ParseProgram()
			if errors := parser.Errors(); len(errors) > 0 {
				relative, _ := filepath.Rel(repoRoot, path)
				t.Errorf("%s Joss block %d does not parse: %s", filepath.ToSlash(relative), index+1, strings.Join(errors, "; "))
			}
		}
	}
}

func TestDocumentedControlFlowParses(t *testing.T) {
	examples := map[string]string{
		"foreach":  "foreach ($items as $item) {\nprint($item)\n}\n",
		"while":    "while ($pending > 0) {\n$pending++\n}\n",
		"do":       "do {\n$attempts++\n} while ($attempts < 3)\n",
		"try":      "try {\nthrow \"fallo\"\n} catch ($error) {\nprint($error)\n}\n",
		"combined": "foreach ($items as $item) {\nprint($item)\n}\nwhile ($pending > 0) {\n$pending++\n}\ndo {\n$attempts++\n} while ($attempts < 3)\ntry {\nthrow \"fallo\"\n} catch ($error) {\nprint($error)\n}\n",
	}
	for name, source := range examples {
		t.Run(name, func(t *testing.T) {
			parser := NewParser(NewLexer(source))
			parser.ParseProgram()
			if errors := parser.Errors(); len(errors) > 0 {
				t.Fatalf("%s", strings.Join(errors, "; "))
			}
		})
	}
}
