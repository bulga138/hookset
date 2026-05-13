package scanner

import (
	"os"
	"path/filepath"

	"github.com/bulga138/hookset/internal/manifest"
)

// Scan looks at the project root and returns suggested hook entries.
func Scan(root string) []manifest.Entry {
	var suggestions []manifest.Entry

	// Node.js
	if fileExists(filepath.Join(root, "package.json")) {
		suggestions = append(suggestions, manifest.Entry{
			Name:    "eslint",
			Event:   "pre-commit",
			Match:   []string{"*.js", "*.ts", "*.jsx", "*.tsx"},
			Command: "npx eslint --fix",
		})
		suggestions = append(suggestions, manifest.Entry{
			Name:    "prettier",
			Event:   "pre-commit",
			Match:   []string{"*.js", "*.ts", "*.css", "*.json", "*.md"},
			Command: "npx prettier --write",
		})
	}

	// Go
	if fileExists(filepath.Join(root, "go.mod")) {
		suggestions = append(suggestions, manifest.Entry{
			Name:    "go-fmt",
			Event:   "pre-commit",
			Match:   []string{"*.go"},
			Command: "go fmt",
		})
		suggestions = append(suggestions, manifest.Entry{
			Name:    "go-test",
			Event:   "pre-push",
			Command: "go test ./...",
		})
	}

	// Python
	if fileExists(filepath.Join(root, "requirements.txt")) || fileExists(filepath.Join(root, "pyproject.toml")) {
		suggestions = append(suggestions, manifest.Entry{
			Name:    "black",
			Event:   "pre-commit",
			Match:   []string{"*.py"},
			Command: "black",
		})
	}

	return suggestions
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
