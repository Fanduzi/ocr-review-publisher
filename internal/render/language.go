package render

import "strings"

// FenceLanguage returns the code fence language for a file path.
// Extension match is case-insensitive. Unknown extensions return "text".
func FenceLanguage(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx < 0 {
		return "text"
	}
	ext := strings.ToLower(path[idx:])
	switch ext {
	case ".go":
		return "go"
	case ".js", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".rs":
		return "rust"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".md":
		return "markdown"
	default:
		return "text"
	}
}
