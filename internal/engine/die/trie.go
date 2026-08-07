package die

import (
	"sort"
	"strings"
)

// TrieNode is a node in a radix trie for O(k) prefix lookups.
type TrieNode struct {
	children map[rune]*TrieNode
	isEnd    bool
	command  string
	title    string
	category string
}

// CommandTrie provides O(k) prefix-based autocomplete.
type CommandTrie struct {
	root *TrieNode
	size int
}

// NewCommandTrie creates an empty Trie.
func NewCommandTrie() *CommandTrie {
	return &CommandTrie{
		root: &TrieNode{children: make(map[rune]*TrieNode)},
	}
}

// Insert adds a command to the Trie with metadata.
func (t *CommandTrie) Insert(cmd, title, category string) {
	node := t.root
	for _, ch := range strings.ToLower(cmd) {
		if _, exists := node.children[ch]; !exists {
			node.children[ch] = &TrieNode{children: make(map[rune]*TrieNode)}
		}
		node = node.children[ch]
	}
	if !node.isEnd {
		t.size++
	}
	node.isEnd = true
	node.command = cmd
	node.title = title
	node.category = category
}

// SearchPrefix returns all commands matching the given prefix.
func (t *CommandTrie) SearchPrefix(prefix string) []Suggestion {
	node := t.root
	for _, ch := range strings.ToLower(prefix) {
		if next, exists := node.children[ch]; exists {
			node = next
		} else {
			return nil
		}
	}

	var results []Suggestion
	t.collect(node, &results, prefix)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > 10 {
		results = results[:10]
	}
	return results
}

func (t *CommandTrie) collect(node *TrieNode, results *[]Suggestion, prefix string) {
	if node.isEnd {
		score := float64(len(prefix)) / float64(len(node.command))
		if score > 1.0 {
			score = 1.0
		}
		*results = append(*results, Suggestion{
			Title:       node.title,
			Description: node.command,
			Category:    node.category,
			Score:       score,
		})
	}
	for _, child := range node.children {
		t.collect(child, results, prefix)
	}
}

// Size returns the number of commands in the Trie.
func (t *CommandTrie) Size() int {
	return t.size
}

// FuzzySearch performs Levenshtein-based fuzzy matching.
func (t *CommandTrie) FuzzySearch(query string, maxDistance int) []Suggestion {
	var results []Suggestion
	t.fuzzyCollect(t.root, "", []rune(strings.ToLower(query)), maxDistance, &results)

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > 8 {
		results = results[:8]
	}
	return results
}

func (t *CommandTrie) fuzzyCollect(node *TrieNode, current string, query []rune, maxDist int, results *[]Suggestion) {
	if node.isEnd {
		dist := levenshtein(strings.ToLower(current), query)
		if dist <= maxDist {
			score := 1.0 - float64(dist)/float64(max(len(current), len(query)))
			*results = append(*results, Suggestion{
				Title:       node.title,
				Description: node.command,
				Category:    node.category,
				Score:       score,
			})
		}
	}

	for ch, child := range node.children {
		t.fuzzyCollect(child, current+string(ch), query, maxDist, results)
	}
}

// levenshtein computes the edit distance between two strings.
func levenshtein(s1 string, s2 []rune) int {
	r1 := []rune(s1)
	len1, len2 := len(r1), len(s2)

	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	// Use single-row DP for memory efficiency
	prev := make([]int, len2+1)
	curr := make([]int, len2+1)

	for j := 0; j <= len2; j++ {
		prev[j] = j
	}

	for i := 1; i <= len1; i++ {
		curr[0] = i
		for j := 1; j <= len2; j++ {
			cost := 1
			if r1[i-1] == s2[j-1] {
				cost = 0
			}
			curr[j] = min3(
				prev[j]+1,      // deletion
				curr[j-1]+1,    // insertion
				prev[j-1]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}

	return prev[len2]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// BuildDefaultTrie creates a Trie pre-populated with all known commands.
func BuildDefaultTrie() *CommandTrie {
	trie := NewCommandTrie()

	commands := []struct {
		cmd      string
		title    string
		category string
	}{
		{"find files", "Find Files", "Search"},
		{"find by extension", "Find by Extension", "Search"},
		{"find by size", "Find by Size", "Search"},
		{"find hidden files", "Find Hidden Files", "Search"},
		{"find duplicate files", "Find Duplicate Files", "Search"},
		{"find empty folders", "Find Empty Folders", "Search"},
		{"find large files", "Find Large Files", "Search"},
		{"find small files", "Find Small Files", "Search"},
		{"find recent files", "Find Recent Files", "Search"},
		{"find old files", "Find Old Files", "Search"},
		{"find images", "Find Images", "Search"},
		{"find videos", "Find Videos", "Search"},
		{"find documents", "Find Documents", "Search"},
		{"find archives", "Find Archives", "Search"},
		{"find executables", "Find Executables", "Search"},
		{"find logs", "Find Log Files", "Search"},
		{"find config files", "Find Config Files", "Search"},
		{"find certificates", "Find Certificates", "Search"},
		{"find ssh keys", "Find SSH Keys", "Search"},
		{"find databases", "Find Databases", "Search"},
		{"show largest files", "Show Largest Files", "Analyze"},
		{"show largest folders", "Show Largest Folders", "Analyze"},
		{"show smallest files", "Show Smallest Files", "Analyze"},
		{"show users", "Show Users", "Analyze"},
		{"show partitions", "Show Partitions", "Analyze"},
		{"show disk info", "Show Disk Info", "Analyze"},
		{"show file statistics", "Show File Statistics", "Analyze"},
		{"show filesystem info", "Show Filesystem Info", "Analyze"},
		{"show geometry", "Show Disk Geometry", "Analyze"},
		{"show container info", "Show Container Info", "Analyze"},
		{"list partitions", "List Partitions", "Analyze"},
		{"list directories", "List Directories", "Analyze"},
		{"list root", "List Root Directory", "Analyze"},
		{"open folder", "Open Folder", "Explorer"},
		{"open path", "Open Path", "Explorer"},
		{"navigate to", "Navigate To", "Explorer"},
		{"go back", "Go Back", "Explorer"},
		{"go forward", "Go Forward", "Explorer"},
		{"refresh", "Refresh View", "Explorer"},
		{"copy path", "Copy Path", "Explorer"},
		{"copy filename", "Copy Filename", "Explorer"},
		{"show properties", "Show Properties", "Explorer"},
		{"calculate hash", "Calculate Hash", "Hash"},
		{"hash file", "Hash File", "Hash"},
		{"md5", "MD5 Hash", "Hash"},
		{"sha256", "SHA-256 Hash", "Hash"},
		{"verify checksum", "Verify Checksum", "Hash"},
		{"extract files", "Extract Files", "Extract"},
		{"extract current", "Extract Current", "Extract"},
		{"extract selected", "Extract Selected", "Extract"},
		{"extract folder", "Extract Folder", "Extract"},
		{"export csv", "Export as CSV", "Export"},
		{"export json", "Export as JSON", "Export"},
		{"export html", "Export as HTML", "Export"},
		{"export report", "Export Report", "Export"},
		{"preview file", "Preview File", "Preview"},
		{"hex view", "Hex View", "Preview"},
		{"text view", "Text View", "Preview"},
		{"image view", "Image View", "Preview"},
		{"compare partitions", "Compare Partitions", "Compare"},
		{"compare folders", "Compare Folders", "Compare"},
		{"compare backups", "Compare Backups", "Compare"},
		{"compare hashes", "Compare Hashes", "Compare"},
		{"generate report", "Generate Report", "Report"},
		{"disk summary", "Disk Summary", "Report"},
		{"file statistics", "File Statistics", "Report"},
		{"recover deleted", "Recover Deleted Files", "Recovery"},
		{"recover folder", "Recover Folder", "Recovery"},
		{"settings", "Open Settings", "Settings"},
		{"open settings", "Open Settings", "Settings"},
	}

	for _, cmd := range commands {
		trie.Insert(cmd.cmd, cmd.title, cmd.category)
	}

	return trie
}
