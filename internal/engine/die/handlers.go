package die

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// HandlerDeps holds dependencies injected into the DIE handlers.
type HandlerDeps struct {
	SearchFunc       func(query string, filters map[string]string, path string) ([]map[string]interface{}, error)
	ListDirFunc      func(path string) ([]map[string]interface{}, error)
	GetDiskInfoFunc  func() (interface{}, error)
	GetPartitionsFunc func() ([]map[string]interface{}, error)
	HashFunc         func(path string, algo string) (string, error)
	ExportFunc       func(format string, data interface{}) (string, error)
	PreviewFunc      func(path string) (interface{}, error)
	ExtractFunc      func(paths []string, dest string) (string, error)
}

// RegisterDefaultHandlers registers all built-in DIE commands.
func RegisterDefaultHandlers(reg *Registry, deps *HandlerDeps) {
	reg.Register(CommandRegistration{
		ID: "search", Title: "Find Files",
		Description: "Search for files by name, extension, or pattern",
		Category: "Search", Keywords: []string{"find", "search", "locate"},
		Action: ActionSearch,
		Handler: func(ctx context.Context, intent Intent, cmdCtx CommandContext) (interface{}, error) {
			if deps.SearchFunc == nil {
				return nil, fmt.Errorf("search not available")
			}
			results, err := deps.SearchFunc(intent.Query, intent.Filters, intent.Target)
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{"action": "search_results", "results": results, "count": len(results)}, nil
		},
	})

	reg.Register(CommandRegistration{
		ID: "navigate", Title: "Open Folder",
		Description: "Navigate to a directory path",
		Category: "Explorer", Keywords: []string{"open", "cd", "navigate", "goto"},
		Action: ActionNavigate,
		Handler: func(ctx context.Context, intent Intent, cmdCtx CommandContext) (interface{}, error) {
			if deps.ListDirFunc == nil {
				return nil, fmt.Errorf("navigation not available")
			}
			target := intent.Target
			if target == "" {
				target = "/"
			}
			entries, err := deps.ListDirFunc(target)
			if err != nil {
				return nil, fmt.Errorf("cannot open %q: %w", target, err)
			}
			return map[string]interface{}{"action": "navigate", "path": target, "entries": entries}, nil
		},
	})

	reg.Register(CommandRegistration{
		ID: "analyze_largest", Title: "Show Largest Files",
		Description: "Find and display the largest files on disk",
		Category: "Analyze", Keywords: []string{"largest", "biggest", "size"},
		Action: ActionAnalyze,
		Handler: func(ctx context.Context, intent Intent, cmdCtx CommandContext) (interface{}, error) {
			if intent.Target == "users" {
				return handleShowUsers(ctx, intent, cmdCtx, deps)
			}
			if intent.Target == "partitions" {
				return handleShowPartitions(ctx, intent, cmdCtx, deps)
			}
			return handleLargestFiles(ctx, intent, cmdCtx, deps)
		},
	})

	reg.Register(CommandRegistration{
		ID: "analyze_disk_info", Title: "Show Disk Info",
		Description: "Display detailed disk and container information",
		Category: "Analyze", Keywords: []string{"disk", "info", "details", "container"},
		Action: ActionAnalyze,
		Handler: func(ctx context.Context, intent Intent, cmdCtx CommandContext) (interface{}, error) {
			if deps.GetDiskInfoFunc == nil {
				return nil, fmt.Errorf("disk info not available")
			}
			info, err := deps.GetDiskInfoFunc()
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{"action": "disk_info", "data": info}, nil
		},
	})

	reg.Register(CommandRegistration{
		ID: "hash", Title: "Calculate Hash",
		Description: "Compute MD5 or SHA-256 hash of a file",
		Category: "Hash", Keywords: []string{"hash", "checksum", "md5", "sha"},
		Action: ActionHash,
		Handler: func(ctx context.Context, intent Intent, cmdCtx CommandContext) (interface{}, error) {
			if deps.HashFunc == nil {
				return nil, fmt.Errorf("hash not available")
			}
			target := intent.Target
			if target == "" {
				target = cmdCtx.SelectedFile
			}
			if target == "" {
				return nil, fmt.Errorf("no file selected")
			}
			algo := "sha256"
			hash, err := deps.HashFunc(target, algo)
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{"action": "hash", "file": target, "algorithm": algo, "hash": hash}, nil
		},
	})

	reg.Register(CommandRegistration{
		ID: "export", Title: "Export Report",
		Description: "Export analysis report in CSV, JSON, or HTML format",
		Category: "Export", Keywords: []string{"export", "report", "csv", "json"},
		Action: ActionExport,
		Handler: func(ctx context.Context, intent Intent, cmdCtx CommandContext) (interface{}, error) {
			if deps.ExportFunc == nil {
				return nil, fmt.Errorf("export not available")
			}
			format := "json"
			if f, ok := intent.Filters["format"]; ok {
				format = f
			}
			path, err := deps.ExportFunc(format, nil)
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{"action": "export", "format": format, "path": path}, nil
		},
	})

	reg.Register(CommandRegistration{
		ID: "preview", Title: "Preview File",
		Description: "Preview the contents of a file",
		Category: "Preview", Keywords: []string{"preview", "view", "peek"},
		Action: ActionPreview,
		Handler: func(ctx context.Context, intent Intent, cmdCtx CommandContext) (interface{}, error) {
			if deps.PreviewFunc == nil {
				return nil, fmt.Errorf("preview not available")
			}
			target := intent.Target
			if target == "" {
				target = cmdCtx.SelectedFile
			}
			if target == "" {
				return nil, fmt.Errorf("no file selected")
			}
			data, err := deps.PreviewFunc(target)
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{"action": "preview", "file": target, "data": data}, nil
		},
	})

	reg.Register(CommandRegistration{
		ID: "extract", Title: "Extract Files",
		Description: "Extract files from the current volume",
		Category: "Extract", Keywords: []string{"extract", "copy", "save"},
		Action: ActionExtract,
		Handler: func(ctx context.Context, intent Intent, cmdCtx CommandContext) (interface{}, error) {
			if deps.ExtractFunc == nil {
				return nil, fmt.Errorf("extraction not available")
			}
			paths := cmdCtx.SelectedFiles
			if len(paths) == 0 && cmdCtx.SelectedFile != "" {
				paths = []string{cmdCtx.SelectedFile}
			}
			if len(paths) == 0 {
				paths = []string{intent.Target}
			}
			dest := filepath.Join(".", "extracted")
			result, err := deps.ExtractFunc(paths, dest)
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{"action": "extract", "source": paths, "destination": dest, "result": result}, nil
		},
	})

	reg.Register(CommandRegistration{
		ID: "compare", Title: "Compare",
		Description: "Compare partitions, folders, or files",
		Category: "Compare", Keywords: []string{"compare", "diff"},
		Action: ActionCompare,
		Handler: func(ctx context.Context, intent Intent, cmdCtx CommandContext) (interface{}, error) {
			return map[string]interface{}{"action": "compare", "message": "Compare requires two targets"}, nil
		},
	})

	reg.Register(CommandRegistration{
		ID: "settings", Title: "Settings",
		Description: "Open application settings",
		Category: "Settings", Keywords: []string{"settings", "config", "preferences"},
		Action: ActionSettings,
		Handler: func(ctx context.Context, intent Intent, cmdCtx CommandContext) (interface{}, error) {
			return map[string]interface{}{"action": "open_settings"}, nil
		},
	})
}

func handleLargestFiles(ctx context.Context, intent Intent, cmdCtx CommandContext, deps *HandlerDeps) (interface{}, error) {
	if deps.SearchFunc == nil {
		return nil, fmt.Errorf("search not available")
	}
	filters := map[string]string{"sort": "size", "order": "desc", "limit": "20"}
	results, err := deps.SearchFunc("*", filters, intent.Target)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"action": "largest_files", "results": results, "count": len(results)}, nil
}

func handleShowUsers(ctx context.Context, intent Intent, cmdCtx CommandContext, deps *HandlerDeps) (interface{}, error) {
	if deps.SearchFunc == nil {
		return nil, fmt.Errorf("search not available")
	}
	results, err := deps.SearchFunc("", map[string]string{"type": "directory"}, "/")
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"action": "show_users", "users": results}, nil
}

func handleShowPartitions(ctx context.Context, intent Intent, cmdCtx CommandContext, deps *HandlerDeps) (interface{}, error) {
	if deps.GetPartitionsFunc == nil {
		return nil, fmt.Errorf("partition info not available")
	}
	partitions, err := deps.GetPartitionsFunc()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"action": "show_partitions", "partitions": partitions}, nil
}

func IntentToString(intent *Intent) string {
	var parts []string
	parts = append(parts, string(intent.Action))
	if intent.Query != "" {
		parts = append(parts, fmt.Sprintf("query=%q", intent.Query))
	}
	if intent.Target != "" {
		parts = append(parts, fmt.Sprintf("target=%q", intent.Target))
	}
	return strings.Join(parts, " ")
}
