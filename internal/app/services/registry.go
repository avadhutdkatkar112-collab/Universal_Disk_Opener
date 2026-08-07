package services

// languageName maps a file extension to the GitHub Linguist canonical language name.
// This is a lightweight subset of the 10,000+ extensions in linguist/languages.yml.
// When an extension is not found here, the GrammarService falls back to the CDN.
var languageName = map[string]string{
	// ── Programming Languages ──────────────────────────────────────────────
	"py":     "Python",
	"pyw":    "Python",
	"pyi":    "Python",
	"gyp":    "Python",
	"js":     "JavaScript",
	"mjs":    "JavaScript",
	"cjs":    "JavaScript",
	"ts":     "TypeScript",
	"tsx":    "TypeScript",
	"mts":    "TypeScript",
	"cts":    "TypeScript",
	"rs":     "Rust",
	"go":     "Go",
	"java":   "Java",
	"kt":     "Kotlin",
	"kts":    "Kotlin",
	"scala":  "Scala",
	"rb":     "Ruby",
	"php":    "PHP",
	"c":      "C",
	"h":      "C",
	"cpp":    "C++",
	"cxx":    "C++",
	"cc":     "C++",
	"c++":    "C++",
	"hpp":    "C++",
	"cs":     "C#",
	"fs":     "F#",
	"swift":  "Swift",
	"m":      "Objective-C",
	"mm":     "Objective-C",
	"clj":    "Clojure",
	"cljs":   "Clojure",
	"hs":     "Haskell",
	"lhs":    "Haskell",
	"lua":    "Lua",
	"r":      "R",
	"rmd":    "R",
	"jl":     "Julia",
	"ex":     "Elixir",
	"exs":    "Elixir",
	"erl":    "Erlang",
	"hrl":    "Erlang",
	"dart":   "Dart",
	"zig":    "Zig",
	"nim":    "Nim",
	"cr":     "Crystal",
	"d":      "D",
	"groovy": "Groovy",
	"sol":    "Solidity",
	"v":      "Verilog",
	"vhd":    "VHDL",
	"vhdl":   "VHDL",
	"sv":     "SystemVerilog",
	"svh":    "SystemVerilog",

	// ── Shell / Scripting ──────────────────────────────────────────────────
	"sh":      "Shell",
	"bash":    "Shell",
	"zsh":     "Shell",
	"ksh":     "Shell",
	"csh":     "Shell",
	"fish":    "Fish",
	"bat":     "Batchfile",
	"cmd":     "Batchfile",
	"ps1":     "PowerShell",
	"psm1":    "PowerShell",
	"psd1":    "PowerShell",
	"vim":     "Vim Script",
	"el":      "Emacs Lisp",

	// ── Web / Markup ───────────────────────────────────────────────────────
	"html":    "HTML",
	"htm":     "HTML",
	"xhtml":   "HTML",
	"xml":     "XML",
	"svg":     "XML",
	"xsl":     "XSLT",
	"xslt":    "XSLT",
	"css":     "CSS",
	"scss":    "SCSS",
	"sass":    "Sass",
	"less":    "Less",
	"vue":     "Vue",
	"svelte":  "Svelte",
	"astro":   "Astro",
	"jsx":     "JavaScript",
	"md":      "Markdown",
	"rst":     "reStructuredText",
	"tex":     "TeX",
	"latex":   "LaTeX",
	"bib":     "BibTeX",

	// ── Config / Data ──────────────────────────────────────────────────────
	"json":    "JSON",
	"jsonl":   "JSON",
	"json5":   "JSON5",
	"jsonc":   "JSON with Comments",
	"yaml":    "YAML",
	"yml":     "YAML",
	"toml":    "TOML",
	"ini":     "INI",
	"cfg":     "INI",
	"conf":    "INI",
	"hcl":     "HCL",
	"tf":      "HCL",
	"tfvars":  "HCL",
	"proto":   "Protocol Buffer",
	"graphql": "GraphQL",
	"gql":     "GraphQL",

	// ── Database ───────────────────────────────────────────────────────────
	"sql":     "SQL",
	"mysql":   "SQL",
	"pgsql":   "SQL",
	"plsql":   "PL/pgSQL",
	"sqlite":  "SQL",
	"cql":     "SQL",

	// ── DevOps / Build ─────────────────────────────────────────────────────
	"dockerfile": "Dockerfile",
	"cmake":      "CMake",
	"makefile":   "Makefile",
	"make":       "Makefile",
	"gradle":     "Groovy",
	"sbt":        "Scala",
	"nginx":      "Nginx",
	"apache":     "ApacheConf",

	// ── Build / Project Files ──────────────────────────────────────────────
	"csproj":     "XML",
	"vbproj":     "XML",
	"fsproj":     "XML",
	"vcxproj":    "XML",
	"sln":        "XML",
	"props":      "XML",
	"targets":    "XML",
	"plist":      "XML",
	"pbxproj":    "Objective-C",

	// ── Documentation ──────────────────────────────────────────────────────
	"adoc":     "AsciiDoc",
	"asciidoc": "AsciiDoc",
	"wiki":     "WikiText",

	// ── Assembly ───────────────────────────────────────────────────────────
	"asm":      "Assembly",
	"s":        "Assembly",
	"nasm":     "Assembly",
	"masm":     "Assembly",

	// ── Scientific / Math ──────────────────────────────────────────────────
	"matlab":   "MATLAB",
	"oct":      "Octave",
	"f90":      "Fortran",
	"f95":      "Fortran",
	"f03":      "Fortran",

	// ── Other Languages ────────────────────────────────────────────────────
	"coffee":    "CoffeeScript",
	"litcoffee": "CoffeeScript",
	"elm":       "Elm",
	"purs":      "PureScript",
	"nix":       "Nix",
	"hx":        "Haxe",

	// ── WebAssembly / Low-level ────────────────────────────────────────────
	"wat":      "WebAssembly",
	"wasm":     "WebAssembly Text",

	// ── Data / Structured ──────────────────────────────────────────────────
	"csv":      "CSV",
	"tsv":      "CSV",
	"parquet":  "Parquet",

	// ── Text / Config Formats ──────────────────────────────────────────────
	"env":           "Dotenv",
	"gitignore":     "Git Ignore",
	"dockerignore":  "Docker Ignore",
	"editorconfig":  "EditorConfig",
	"sshconfig":     "SSH Config",
	"sshdconfig":    "SSHD Config",

	// ── Backup / Temp ──────────────────────────────────────────────────────
	"save":    "JavaScript",
	"bak":     "JavaScript",
	"old":     "JavaScript",
	"orig":    "JavaScript",
	"tmp":     "Text",
	"temp":    "Text",
	"swp":     "Text",
	"swo":     "Text",
	"backup":  "Text",

	// ── Game / Graphics ────────────────────────────────────────────────────
	"glsl":     "GLSL",
	"hlsl":     "HLSL",
	"wgsl":     "WGSL",
	"frag":     "GLSL",
	"vert":     "GLSL",
	"geom":     "GLSL",
}

// LanguageForExt returns the GitHub Linguist canonical language name for a file extension.
// Returns empty string if unknown.
func LanguageForExt(ext string) string {
	return languageName[ext]
}
