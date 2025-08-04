// Package discovery handles command discovery functionality
package discovery

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/mjmorales/simple-mcp-runner/internal/config"
	apperrors "github.com/mjmorales/simple-mcp-runner/internal/errors"
	"github.com/mjmorales/simple-mcp-runner/internal/logger"
	"github.com/mjmorales/simple-mcp-runner/pkg/types"
)

// Discoverer handles command discovery.
type Discoverer struct {
	config *config.Config
	logger *logger.Logger
	cache  *discoveryCache
}

// discoveryCache caches discovery results.
type discoveryCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
}

type cacheEntry struct {
	commands []types.CommandInfo
	paths    []string
}

// New creates a new discoverer instance.
func New(cfg *config.Config, log *logger.Logger) *Discoverer {
	return &Discoverer{
		config: cfg,
		logger: log,
		cache: &discoveryCache{
			entries: make(map[string]*cacheEntry),
		},
	}
}

// Discover finds commands based on the request parameters.
func (d *Discoverer) Discover(ctx context.Context, req *types.CommandDiscoveryRequest) (*types.CommandDiscoveryResult, error) {
	// Set defaults
	if req.Pattern == "" {
		req.Pattern = "*"
	}

	if req.MaxResults <= 0 {
		req.MaxResults = d.config.Discovery.MaxResults
		if req.MaxResults <= 0 {
			req.MaxResults = 100
		}
	}

	// Check cache
	cacheKey := d.getCacheKey(req)
	if cached := d.cache.get(cacheKey); cached != nil {
		return d.buildResult(cached.commands, cached.paths, req.MaxResults), nil
	}

	// Get search paths
	searchPaths := d.getSearchPaths(req)

	// Discover commands
	commands, err := d.discoverInPaths(ctx, searchPaths, req)
	if err != nil {
		return nil, err
	}

	// Sort by relevance
	d.sortCommands(commands, req.Pattern)

	// Cache results
	d.cache.set(cacheKey, &cacheEntry{
		commands: commands,
		paths:    searchPaths,
	})

	return d.buildResult(commands, searchPaths, req.MaxResults), nil
}

// getSearchPaths returns the paths to search for commands.
func (d *Discoverer) getSearchPaths(req *types.CommandDiscoveryRequest) []string {
	pathSet := make(map[string]bool)

	// Add system PATH
	if pathEnv := os.Getenv("PATH"); pathEnv != "" {
		for _, p := range filepath.SplitList(pathEnv) {
			if p != "" && !d.isExcludedPath(p) {
				pathSet[p] = true
			}
		}
	}

	// Add configured additional paths
	for _, p := range d.config.Discovery.AdditionalPaths {
		if !d.isExcludedPath(p) {
			pathSet[p] = true
		}
	}

	// Add request-specific paths
	for _, p := range req.Paths {
		if !d.isExcludedPath(p) {
			pathSet[p] = true
		}
	}

	// Convert to slice
	paths := make([]string, 0, len(pathSet))
	for p := range pathSet {
		paths = append(paths, p)
	}

	sort.Strings(paths)
	return paths
}

// isExcludedPath checks if a path should be excluded.
func (d *Discoverer) isExcludedPath(path string) bool {
	for _, excluded := range d.config.Discovery.ExcludePaths {
		if path == excluded || strings.HasPrefix(path, excluded+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// discoverInPaths discovers commands in the given paths.
func (d *Discoverer) discoverInPaths(ctx context.Context, paths []string, req *types.CommandDiscoveryRequest) ([]types.CommandInfo, error) {
	var (
		commands []types.CommandInfo
		mu       sync.Mutex
		wg       sync.WaitGroup
		errChan  = make(chan error, len(paths))
	)

	// Use a semaphore to limit concurrent directory reads
	sem := make(chan struct{}, 10)

	for _, path := range paths {
		// Check context
		select {
		case <-ctx.Done():
			return nil, apperrors.TimeoutError("discovery cancelled", "")
		default:
		}

		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			cmds := d.discoverInPath(p, req)

			mu.Lock()
			commands = append(commands, cmds...)
			mu.Unlock()
		}(path)
	}

	wg.Wait()
	close(errChan)

	// Check for any critical errors
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return d.deduplicateCommands(commands), nil
}

// discoverInPath discovers commands in a single path.
func (d *Discoverer) discoverInPath(path string, req *types.CommandDiscoveryRequest) []types.CommandInfo {
	entries, err := os.ReadDir(path)
	if err != nil {
		// Path might not exist or be inaccessible
		return nil
	}

	commands := make([]types.CommandInfo, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip hidden files
		if strings.HasPrefix(name, ".") {
			continue
		}

		// Check pattern match
		if !d.matchesPattern(name, req.Pattern) {
			continue
		}

		fullPath := filepath.Join(path, name)

		// Check if executable
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if !d.isExecutable(info) {
			continue
		}

		cmd := types.CommandInfo{
			Name:       name,
			Path:       fullPath,
			Executable: true,
		}

		// Add description if requested
		if req.IncludeDesc {
			cmd.Description = d.getCommandDescription(name)
		}

		commands = append(commands, cmd)
	}

	return commands
}

// matchesPattern checks if a command name matches the pattern.
func (d *Discoverer) matchesPattern(name, pattern string) bool {
	if pattern == "*" || pattern == "" {
		// For wildcard, only include common commands to avoid overwhelming output
		return d.isCommonCommand(name)
	}

	// Try glob match
	if matched, _ := filepath.Match(pattern, name); matched {
		return true
	}

	// Try substring match
	return strings.Contains(strings.ToLower(name), strings.ToLower(pattern))
}

// isCommonCommand checks if a command is in the common commands list.
func (d *Discoverer) isCommonCommand(name string) bool {
	commonCmds := d.config.Discovery.CommonCommands
	if len(commonCmds) == 0 {
		// Default common commands
		commonCmds = []string{
			"ls", "cat", "grep", "find", "git", "npm", "go",
			"python", "node", "curl", "wget", "echo", "pwd",
			"cp", "mv", "mkdir", "touch", "chmod", "ps", "df",
		}
	}

	baseName := strings.TrimSuffix(name, filepath.Ext(name))

	for _, common := range commonCmds {
		if baseName == common || strings.HasPrefix(baseName, common+"-") {
			return true
		}
	}

	return false
}

// isExecutable checks if a file is executable.
func (d *Discoverer) isExecutable(info os.FileInfo) bool {
	if runtime.GOOS == "windows" {
		// On Windows, check file extension
		name := strings.ToLower(info.Name())
		exts := []string{".exe", ".cmd", ".bat", ".com", ".ps1"}
		for _, ext := range exts {
			if strings.HasSuffix(name, ext) {
				return true
			}
		}
		return false
	}

	// On Unix-like systems, check execute permission
	return info.Mode()&0111 != 0
}

// getCommandDescription returns a description for common commands.
func (d *Discoverer) getCommandDescription(name string) string {
	// Remove extension for lookup
	baseName := strings.TrimSuffix(name, filepath.Ext(name))

	descriptions := map[string]string{
		"ls":        "List directory contents",
		"cat":       "Display file contents",
		"grep":      "Search text patterns in files",
		"find":      "Search for files and directories",
		"git":       "Version control system",
		"npm":       "Node.js package manager",
		"go":        "Go programming language toolchain",
		"python":    "Python interpreter",
		"node":      "Node.js JavaScript runtime",
		"curl":      "Transfer data from/to servers",
		"wget":      "Download files from web",
		"echo":      "Display a line of text",
		"pwd":       "Print working directory",
		"cp":        "Copy files or directories",
		"mv":        "Move/rename files or directories",
		"rm":        "Remove files or directories",
		"mkdir":     "Create directories",
		"touch":     "Create empty files or update timestamps",
		"chmod":     "Change file permissions",
		"ps":        "Display running processes",
		"df":        "Display disk space usage",
		"du":        "Display directory space usage",
		"tar":       "Archive files",
		"zip":       "Compress files",
		"unzip":     "Extract compressed files",
		"ssh":       "Secure shell client",
		"scp":       "Secure copy files",
		"rsync":     "Synchronize files/directories",
		"docker":    "Container platform",
		"kubectl":   "Kubernetes command-line tool",
		"terraform": "Infrastructure as code tool",
		"aws":       "AWS command-line interface",
		"gcloud":    "Google Cloud command-line interface",
		"az":        "Azure command-line interface",
	}

	if desc, ok := descriptions[baseName]; ok {
		return desc
	}

	// Check for common prefixes
	prefixes := map[string]string{
		"git-":    "Git subcommand",
		"npm-":    "NPM subcommand",
		"docker-": "Docker subcommand",
		"aws-":    "AWS CLI subcommand",
	}

	for prefix, desc := range prefixes {
		if strings.HasPrefix(baseName, prefix) {
			return desc
		}
	}

	return "System command"
}

// deduplicateCommands removes duplicate commands, keeping the first occurrence.
func (d *Discoverer) deduplicateCommands(commands []types.CommandInfo) []types.CommandInfo {
	seen := make(map[string]bool)
	result := make([]types.CommandInfo, 0, len(commands))

	for _, cmd := range commands {
		if !seen[cmd.Name] {
			seen[cmd.Name] = true
			result = append(result, cmd)
		}
	}

	return result
}

// sortCommands sorts commands by relevance.
func (d *Discoverer) sortCommands(commands []types.CommandInfo, pattern string) {
	sort.Slice(commands, func(i, j int) bool {
		// Exact matches first
		if commands[i].Name == pattern && commands[j].Name != pattern {
			return true
		}
		if commands[j].Name == pattern && commands[i].Name != pattern {
			return false
		}

		// Common commands before others
		iCommon := d.isCommonCommand(commands[i].Name)
		jCommon := d.isCommonCommand(commands[j].Name)
		if iCommon && !jCommon {
			return true
		}
		if jCommon && !iCommon {
			return false
		}

		// Alphabetical order
		return commands[i].Name < commands[j].Name
	})
}

// buildResult builds the discovery result.
func (d *Discoverer) buildResult(commands []types.CommandInfo, paths []string, maxResults int) *types.CommandDiscoveryResult {
	totalFound := len(commands)
	truncated := false

	if maxResults > 0 && len(commands) > maxResults {
		commands = commands[:maxResults]
		truncated = true
	}

	return &types.CommandDiscoveryResult{
		Commands:    commands,
		TotalFound:  totalFound,
		Truncated:   truncated,
		SearchPaths: paths,
	}
}

// getCacheKey generates a cache key for the request.
func (d *Discoverer) getCacheKey(req *types.CommandDiscoveryRequest) string {
	parts := []string{
		req.Pattern,
		strings.Join(req.Paths, "|"),
	}
	return strings.Join(parts, ":")
}

// Cache methods

func (c *discoveryCache) get(key string) *cacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.entries[key]
}

func (c *discoveryCache) set(key string, entry *cacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple cache eviction - limit to 100 entries
	if len(c.entries) >= 100 {
		// Remove a random entry
		for k := range c.entries {
			delete(c.entries, k)
			break
		}
	}

	c.entries[key] = entry
}

// Clear clears the discovery cache.
func (d *Discoverer) ClearCache() {
	d.cache.mu.Lock()
	defer d.cache.mu.Unlock()
	d.cache.entries = make(map[string]*cacheEntry)
}
