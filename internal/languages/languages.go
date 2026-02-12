package languages

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/css"
	"github.com/smacker/go-tree-sitter/cue"
	"github.com/smacker/go-tree-sitter/dockerfile"
	"github.com/smacker/go-tree-sitter/elixir"
	"github.com/smacker/go-tree-sitter/elm"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/groovy"
	"github.com/smacker/go-tree-sitter/hcl"
	"github.com/smacker/go-tree-sitter/html"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/kotlin"
	"github.com/smacker/go-tree-sitter/lua"
	"github.com/smacker/go-tree-sitter/ocaml"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/protobuf"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/scala"
	"github.com/smacker/go-tree-sitter/sql"
	"github.com/smacker/go-tree-sitter/svelte"
	"github.com/smacker/go-tree-sitter/swift"
	"github.com/smacker/go-tree-sitter/toml"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
	"github.com/smacker/go-tree-sitter/yaml"
)

// LanguageConfig holds information about a supported language
type LanguageConfig struct {
	Extensions []string
	Language   *sitter.Language
}

// SupportedLanguages maps language names to their configurations
var SupportedLanguages = map[string]LanguageConfig{
	"bash": {
		Extensions: []string{".sh", ".bash"},
		Language:   bash.GetLanguage(),
	},
	"c": {
		Extensions: []string{".c", ".h"},
		Language:   c.GetLanguage(),
	},
	"cpp": {
		Extensions: []string{".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx"},
		Language:   cpp.GetLanguage(),
	},
	"csharp": {
		Extensions: []string{".cs"},
		Language:   csharp.GetLanguage(),
	},
	"css": {
		Extensions: []string{".css"},
		Language:   css.GetLanguage(),
	},
	"cue": {
		Extensions: []string{".cue"},
		Language:   cue.GetLanguage(),
	},
	"dockerfile": {
		Extensions: []string{"Dockerfile"},
		Language:   dockerfile.GetLanguage(),
	},
	"elixir": {
		Extensions: []string{".ex", ".exs"},
		Language:   elixir.GetLanguage(),
	},
	"elm": {
		Extensions: []string{".elm"},
		Language:   elm.GetLanguage(),
	},
	"go": {
		Extensions: []string{".go"},
		Language:   golang.GetLanguage(),
	},
	"groovy": {
		Extensions: []string{".groovy", ".gradle"},
		Language:   groovy.GetLanguage(),
	},
	"hcl": {
		Extensions: []string{".hcl", ".tf", ".tfvars"},
		Language:   hcl.GetLanguage(),
	},
	"html": {
		Extensions: []string{".html", ".htm"},
		Language:   html.GetLanguage(),
	},
	"java": {
		Extensions: []string{".java"},
		Language:   java.GetLanguage(),
	},
	"javascript": {
		Extensions: []string{".js", ".jsx", ".mjs", ".cjs"},
		Language:   javascript.GetLanguage(),
	},
	"kotlin": {
		Extensions: []string{".kt", ".kts"},
		Language:   kotlin.GetLanguage(),
	},
	"lua": {
		Extensions: []string{".lua"},
		Language:   lua.GetLanguage(),
	},
	"ocaml": {
		Extensions: []string{".ml", ".mli"},
		Language:   ocaml.GetLanguage(),
	},
	"php": {
		Extensions: []string{".php"},
		Language:   php.GetLanguage(),
	},
	"protobuf": {
		Extensions: []string{".proto"},
		Language:   protobuf.GetLanguage(),
	},
	"python": {
		Extensions: []string{".py", ".pyi"},
		Language:   python.GetLanguage(),
	},
	"ruby": {
		Extensions: []string{".rb", ".rake", ".gemspec"},
		Language:   ruby.GetLanguage(),
	},
	"rust": {
		Extensions: []string{".rs"},
		Language:   rust.GetLanguage(),
	},
	"scala": {
		Extensions: []string{".scala", ".sc"},
		Language:   scala.GetLanguage(),
	},
	"sql": {
		Extensions: []string{".sql"},
		Language:   sql.GetLanguage(),
	},
	"svelte": {
		Extensions: []string{".svelte"},
		Language:   svelte.GetLanguage(),
	},
	"swift": {
		Extensions: []string{".swift"},
		Language:   swift.GetLanguage(),
	},
	"toml": {
		Extensions: []string{".toml"},
		Language:   toml.GetLanguage(),
	},
	"tsx": {
		Extensions: []string{".tsx"},
		Language:   tsx.GetLanguage(),
	},
	"typescript": {
		Extensions: []string{".ts", ".mts", ".cts"},
		Language:   typescript.GetLanguage(),
	},
	"yaml": {
		Extensions: []string{".yaml", ".yml"},
		Language:   yaml.GetLanguage(),
	},
}

// extensionToLanguage maps file extensions to their language configuration
var extensionToLanguage map[string]*LanguageConfig

func init() {
	extensionToLanguage = make(map[string]*LanguageConfig)
	for langName := range SupportedLanguages {
		config := SupportedLanguages[langName]
		for _, ext := range config.Extensions {
			extensionToLanguage[ext] = &config
		}
	}
}

// IsSupportedExtension checks if a file extension is supported
func IsSupportedExtension(ext string) bool {
	_, ok := extensionToLanguage[ext]
	return ok
}

// GetLanguageForExtension returns the tree-sitter language for a file extension
func GetLanguageForExtension(ext string) *sitter.Language {
	if config, ok := extensionToLanguage[ext]; ok {
		return config.Language
	}
	return nil
}

// GetSupportedExtensions returns a list of all supported file extensions
func GetSupportedExtensions() []string {
	extensions := make([]string, 0, len(extensionToLanguage))
	for ext := range extensionToLanguage {
		extensions = append(extensions, ext)
	}
	return extensions
}

// GetSupportedLanguageNames returns a sorted list of all supported language names
func GetSupportedLanguageNames() []string {
	names := make([]string, 0, len(SupportedLanguages))
	for name := range SupportedLanguages {
		names = append(names, name)
	}
	// Sort alphabetically
	for i := 0; i < len(names)-1; i++ {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	return names
}
