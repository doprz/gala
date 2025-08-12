package main

import (
	"slices"
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/charmbracelet/lipgloss"
	"github.com/olekukonko/tablewriter"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// Version and build info
const (
	Version     = "1.0.0"
	AppName     = "Gala"
	Description = "Git Author Line Analyzer - Analyzes git blame data to show author contributions"
)

// Styles for consistent UI
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")). // Cyan
			Render

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")). // Blue
			Render

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")). // Green
			Render

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // Yellow
			Render

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")). // Red
			Render

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")). // Gray
			Render
)

// Config holds application configuration
type Config struct {
	Directory   string
	Username    string
	Concurrency int
	ShowHelp    bool
	Verbose     bool
}

// AuthorStats represents statistics for an author
type AuthorStats struct {
	Name      string
	LineCount int
}

// FileContribution represents a file contribution by a user
type FileContribution struct {
	Path      string
	LineCount int
}

// AnalysisResult holds the results of git analysis
type AnalysisResult struct {
	Authors           []AuthorStats
	UserContributions []FileContribution
	TotalLines        int
	FilesProcessed    int
	TotalFiles        int
}

// GitAnalyzer handles git repository analysis
type GitAnalyzer struct {
	config          Config
	excludePatterns []string
	gitignoreGlobs  []string
}

// NewGitAnalyzer creates a new GitAnalyzer instance
func NewGitAnalyzer(config Config) *GitAnalyzer {
	return &GitAnalyzer{
		config:          config,
		excludePatterns: getDefaultExcludePatterns(),
	}
}

// getDefaultExcludePatterns returns default file patterns to exclude
func getDefaultExcludePatterns() []string {
	return []string{
		// Lock files
		"*-lock.*", "*.lock",
		// Images
		"*.gif", "*.png", "*.jpg", "*.jpeg", "*.webp", "*.ico", "*.tiff", "*.tif", "*.bmp", "*.svg",
		// Fonts
		"*.woff", "*.woff2", "*.ttf", "*.otf", "*.eot",
		// Media
		"*.mp4", "*.avi", "*.mov", "*.wmv", "*.flv", "*.webm", "*.mp3", "*.wav", "*.flac", "*.aac", "*.ogg",
		// Archives
		"*.zip", "*.tar", "*.tgz", "*.rar", "*.7z", "*.gz", "*.bz2", "*.xz",
		// Binaries
		"*.exe", "*.dll", "*.so", "*.dylib", "*.bin", "*.deb", "*.rpm", "*.dmg", "*.pkg", "*.msi",
		// Databases
		"*.db", "*.sqlite", "*.sqlite3", "*.mdb",
		// Documents
		"*.pdf", "*.doc", "*.docx", "*.xls", "*.xlsx", "*.ppt", "*.pptx",
		// Compiled
		"*.o", "*.obj", "*.class", "*.pyc", "*.pyo", "*.pyd", "*.a", "*.lib", "*.jar", "*.war", "*.ear",
		// Minified
		"*.min.js", "*.min.css", "*.min.html",
		// OS files
		".DS_Store", "Thumbs.db", "desktop.ini", ".directory",
		// IDE files
		"*.swp", "*.swo", "*~",
		// Logs
		"*.log", "*.logs",
		// Certificates
		"*.pem", "*.key", "*.p12", "*.pfx", "*.crt", "*.cer",
		// Backups
		"*.bak", "*.backup", "*.orig",
	}
}

// validateDirectory checks if the directory exists and is a git repository
func (ga *GitAnalyzer) validateDirectory() error {
	// Check if directory exists
	info, err := os.Stat(ga.config.Directory)
	if err != nil {
		return fmt.Errorf("directory %q does not exist", ga.config.Directory)
	}

	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", ga.config.Directory)
	}

	// Check if it's a git repository
	gitDir := filepath.Join(ga.config.Directory, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return fmt.Errorf("%q is not a git repository", ga.config.Directory)
	}

	return nil
}

// loadGitignorePatterns loads patterns from .gitignore file
func (ga *GitAnalyzer) loadGitignorePatterns() error {
	gitignorePath := filepath.Join(ga.config.Directory, ".gitignore")

	file, err := os.Open(gitignorePath)
	if err != nil {
		// .gitignore doesn't exist, that's okay
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	patterns := make([]string, 0)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Skip negation patterns for simplicity
		if strings.HasPrefix(line, "!") {
			continue
		}

		// Convert gitignore pattern to glob pattern
		pattern := line
		if strings.HasSuffix(pattern, "/") {
			pattern = strings.TrimSuffix(pattern, "/")
		}

		patterns = append(patterns, pattern)
	}

	ga.gitignoreGlobs = patterns
	if len(patterns) > 0 && ga.config.Verbose {
		fmt.Printf("%s Loaded %d patterns from .gitignore\n",
			successStyle("✓"), len(patterns))
	}

	return scanner.Err()
}

// shouldExcludeFile checks if a file should be excluded based on patterns
func (ga *GitAnalyzer) shouldExcludeFile(filePath string) bool {
	fileName := filepath.Base(filePath)

	// Check default exclude patterns
	for _, pattern := range ga.excludePatterns {
		if matched, _ := filepath.Match(pattern, fileName); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, filePath); matched {
			return true
		}
	}

	// Check gitignore patterns
	for _, pattern := range ga.gitignoreGlobs {
		if matched, _ := filepath.Match(pattern, fileName); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, filePath); matched {
			return true
		}
		if strings.Contains(filePath, pattern) {
			return true
		}
	}

	return false
}

// findFiles finds all files to analyze
func (ga *GitAnalyzer) findFiles() ([]string, error) {
	var files []string

	err := filepath.Walk(ga.config.Directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories
		if info.IsDir() {
			// Skip common directories
			dirName := filepath.Base(path)
			skipDirs := []string{
				".git", "node_modules", "vendor", ".cache", "__pycache__",
				".vscode", ".idea", ".vs", "dist", "build",
			}
			if slices.Contains(skipDirs, dirName) {
					return filepath.SkipDir
				}
			return nil
		}

		// Get relative path from target directory
		relPath, err := filepath.Rel(ga.config.Directory, path)
		if err != nil {
			return nil
		}

		// Check if file should be excluded
		if !ga.shouldExcludeFile(relPath) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// BlameResult represents the result of git blame for a file
type BlameResult struct {
	FilePath string
	Authors  []string
	Error    error
}

// runGitBlame runs git blame on a single file
func (ga *GitAnalyzer) runGitBlame(ctx context.Context, filePath string) BlameResult {
	// Convert to relative path for git blame
	relPath, err := filepath.Rel(ga.config.Directory, filePath)
	if err != nil {
		return BlameResult{FilePath: filePath, Error: err}
	}

	// Run git blame with options for better accuracy
	cmd := exec.CommandContext(ctx, "git", "blame", "-M", "-C", "-w", "--line-porcelain", relPath)
	cmd.Dir = ga.config.Directory

	output, err := cmd.Output()
	if err != nil {
		return BlameResult{FilePath: filePath, Error: err}
	}

	// Parse authors from porcelain output
	authors := make([]string, 0)
	lines := strings.SplitSeq(string(output), "\n")

	for line := range lines {
		if strings.HasPrefix(line, "author ") {
			author := strings.TrimPrefix(line, "author ")
			if author != "" {
				authors = append(authors, author)
			}
		}
	}

	return BlameResult{FilePath: filePath, Authors: authors}
}

// processFiles processes files concurrently and returns analysis results
func (ga *GitAnalyzer) processFiles(ctx context.Context, files []string) (*AnalysisResult, error) {
	// Create worker pool
	concurrency := ga.config.Concurrency
	if concurrency <= 0 {
		concurrency = runtime.NumCPU() * 2
	}

	// Create progress bar
	bar := progressbar.NewOptions(len(files),
		progressbar.OptionSetDescription("Processing files..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerPadding: "░",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(50),
	)

	// Results channels
	resultsChan := make(chan BlameResult, len(files))

	// Worker group
	g, ctx := errgroup.WithContext(ctx)

	// Input channel for file paths
	fileChan := make(chan string, len(files))

	// Start workers
	for i := 0; i < concurrency; i++ {
		g.Go(func() error {
			for filePath := range fileChan {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					result := ga.runGitBlame(ctx, filePath)
					resultsChan <- result
					bar.Add(1)
				}
			}
			return nil
		})
	}

	// Send files to workers
	go func() {
		defer close(fileChan)
		for _, file := range files {
			select {
			case fileChan <- file:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Collect results
	go func() {
		g.Wait()
		close(resultsChan)
	}()

	// Process results
	authorCounts := make(map[string]int)
	userContributions := make(map[string]int)
	totalLines := 0
	filesProcessed := 0

	for result := range resultsChan {
		if result.Error != nil {
			if ga.config.Verbose {
				fmt.Printf("%s Error processing %s: %v\n",
					warningStyle("⚠"), result.FilePath, result.Error)
			}
			continue
		}

		filesProcessed++

		// Count authors
		for _, author := range result.Authors {
			if author != "" {
				authorCounts[author]++
				totalLines++

				// If filtering for specific user, track per-file contributions
				if ga.config.Username != "" && author == ga.config.Username {
					relPath, _ := filepath.Rel(ga.config.Directory, result.FilePath)
					userContributions[relPath]++
				}
			}
		}
	}

	bar.Finish()
	fmt.Println() // Add line break after progress bar

	// Check for worker errors
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Convert to sorted slices
	authors := make([]AuthorStats, 0, len(authorCounts))
	for name, count := range authorCounts {
		authors = append(authors, AuthorStats{Name: name, LineCount: count})
	}

	// Sort authors by line count (descending)
	sort.Slice(authors, func(i, j int) bool {
		return authors[i].LineCount > authors[j].LineCount
	})

	// Convert user contributions to sorted slice
	contributions := make([]FileContribution, 0, len(userContributions))
	for path, count := range userContributions {
		contributions = append(contributions, FileContribution{Path: path, LineCount: count})
	}

	// Sort contributions by line count (descending)
	sort.Slice(contributions, func(i, j int) bool {
		return contributions[i].LineCount > contributions[j].LineCount
	})

	return &AnalysisResult{
		Authors:           authors,
		UserContributions: contributions,
		TotalLines:        totalLines,
		FilesProcessed:    filesProcessed,
		TotalFiles:        len(files),
	}, nil
}

// displayResults displays the analysis results
func (ga *GitAnalyzer) displayResults(result *AnalysisResult) {
	if ga.config.Username != "" {
		ga.displayUserResults(result)
	} else {
		ga.displayAuthorResults(result)
	}
}

// displayAuthorResults displays results for all authors
func (ga *GitAnalyzer) displayAuthorResults(result *AnalysisResult) {
	fmt.Printf("\n%s\n", headerStyle("🏆 Author Contributions by Lines"))

	if len(result.Authors) == 0 {
		fmt.Printf("%s No authors found!\n", warningStyle("⚠"))
		return
	}

	// Create table
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Rank", "Lines", "Author"})

	// Show top 20 authors
	displayCount := min(len(result.Authors), 20)

	for i := range displayCount {
		author := result.Authors[i]
		rank := fmt.Sprintf("%d", i+1)

		// Add medals for top 3
		switch i {
		case 0:
			rank = "🥇"
		case 1:
			rank = "🥈"
		case 2:
			rank = "🥉"
		}

		table.Append([]string{
			rank,
			formatNumber(author.LineCount),
			author.Name,
		})
	}

	table.Render()

	if len(result.Authors) > displayCount {
		fmt.Printf("%s ... and %d more authors\n\n",
			dimStyle(""), len(result.Authors)-displayCount)
	}

	// Summary
	ga.displaySummary(result)
}

// displayUserResults displays results for a specific user
func (ga *GitAnalyzer) displayUserResults(result *AnalysisResult) {
	fmt.Printf("\n%s\n", headerStyle(fmt.Sprintf("📝 %s's Contributions by File", ga.config.Username)))

	if len(result.UserContributions) == 0 {
		fmt.Printf("%s No contributions found for user %q\n",
			warningStyle("⚠"), ga.config.Username)
		return
	}

	// Create table
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Lines", "File"})
	// Show top 20 files
	displayCount := min(len(result.UserContributions), 20)

	totalUserLines := 0
	for i := range displayCount {
		contrib := result.UserContributions[i]
		totalUserLines += contrib.LineCount

		table.Append([]string{
			formatNumber(contrib.LineCount),
			contrib.Path,
		})
	}

	// Count total user lines across all files
	for _, contrib := range result.UserContributions {
		if displayCount >= len(result.UserContributions) {
			break
		}
		totalUserLines += contrib.LineCount
	}

	table.Render()

	if len(result.UserContributions) > displayCount {
		fmt.Printf("%s ... and %d more files\n\n",
			dimStyle(""), len(result.UserContributions)-displayCount)
	}

	// User-specific summary
	summaryTable := tablewriter.NewWriter(os.Stdout)
	summaryTable.Header([]string{"Metric", "Value"})

	// Calculate total user lines
	userTotal := 0
	for _, contrib := range result.UserContributions {
		userTotal += contrib.LineCount
	}

	summaryTable.Append([]string{"Total lines by " + ga.config.Username, formatNumber(userTotal)})
	summaryTable.Append([]string{"Files contributed to", formatNumber(len(result.UserContributions))})
	summaryTable.Append([]string{"Files processed", formatNumber(result.FilesProcessed)})

	fmt.Printf("\n%s\n", titleStyle("📊 SUMMARY"))
	summaryTable.Render()
}

// displaySummary displays summary statistics
func (ga *GitAnalyzer) displaySummary(result *AnalysisResult) {
	summaryTable := tablewriter.NewWriter(os.Stdout)
	summaryTable.Header([]string{"Metric", "Value"})
	// summaryTable.SetHeaderColor(
	// 	tablewriter.Colors{tablewriter.FgCyanColor, tablewriter.Bold},
	// 	tablewriter.Colors{tablewriter.FgCyanColor, tablewriter.Bold},
	// )
	// summaryTable.SetBorder(true)

	summaryTable.Append([]string{"Total lines analyzed", formatNumber(result.TotalLines)})
	summaryTable.Append([]string{"Unique authors", formatNumber(len(result.Authors))})
	summaryTable.Append([]string{"Files processed", formatNumber(result.FilesProcessed)})

	fmt.Printf("\n%s\n", titleStyle("📊 SUMMARY"))
	summaryTable.Render()
}

// formatNumber formats a number with commas
func formatNumber(n int) string {
	str := strconv.Itoa(n)
	if len(str) <= 3 {
		return str
	}

	var result strings.Builder
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteString(",")
		}
		result.WriteRune(digit)
	}
	return result.String()
}

// Run executes the analysis
func (ga *GitAnalyzer) Run(ctx context.Context) error {
	// Validate directory
	if err := ga.validateDirectory(); err != nil {
		return err
	}

	// Load gitignore patterns
	if err := ga.loadGitignorePatterns(); err != nil {
		return fmt.Errorf("failed to load .gitignore: %w", err)
	}

	// Find files
	fmt.Printf("%s Scanning directory: %s\n",
		titleStyle("🔍"), ga.config.Directory)

	if ga.config.Username != "" {
		fmt.Printf("%s Analyzing contributions by user: %s\n",
			headerStyle("ℹ"), ga.config.Username)
	}

	files, err := ga.findFiles()
	if err != nil {
		return fmt.Errorf("failed to find files: %w", err)
	}

	fmt.Printf("%s Found %s files to analyze\n",
		successStyle("✓"), successStyle(formatNumber(len(files))))

	if len(files) == 0 {
		fmt.Printf("%s No files found to analyze!\n", warningStyle("⚠"))
		return nil
	}

	// Process files
	fmt.Printf("\n%s\n", headerStyle("📊 Processing Files"))

	result, err := ga.processFiles(ctx, files)
	if err != nil {
		return fmt.Errorf("failed to process files: %w", err)
	}

	// Display results
	ga.displayResults(result)

	return nil
}

// CLI setup
func main() {
	var config Config

	rootCmd := &cobra.Command{
		Use:   "gala [directory] [username]",
		Short: Description,
		Long: fmt.Sprintf(`%s

Analyzes git blame data to show author contributions by line count.

Examples:
  # Show all authors across all files
  gala

  # Analyze specific directory  
  gala /path/to/project

  # Show specific user's contributions per file
  gala . "John Doe"

  # Analyze user in different directory
  gala /path/to/repo alice`, titleStyle(fmt.Sprintf("%s v%s", AppName, Version))),

		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse arguments
			if len(args) >= 1 {
				config.Directory = args[0]
			} else {
				config.Directory = "."
			}

			if len(args) >= 2 {
				config.Username = args[1]
			}

			// Convert to absolute path
			absPath, err := filepath.Abs(config.Directory)
			if err != nil {
				return fmt.Errorf("invalid directory path: %w", err)
			}
			config.Directory = absPath

			// Create analyzer and run
			analyzer := NewGitAnalyzer(config)

			// Setup context with cancellation
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle interruption gracefully
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigChan
				fmt.Printf("\n%s Received interrupt signal, shutting down gracefully...\n",
					warningStyle("⚠"))
				cancel()
			}()

			return analyzer.Run(ctx)
		},
	}

	// Add flags
	rootCmd.Flags().IntVarP(&config.Concurrency, "concurrency", "c", 0,
		"Number of concurrent git blame processes (default: 2*CPU cores)")
	rootCmd.Flags().BoolVarP(&config.Verbose, "verbose", "v", false,
		"Enable verbose output")

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("%s %v\n", errorStyle("✗"), err)
		os.Exit(1)
	}
}
