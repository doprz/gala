package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/olekukonko/tablewriter"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

// Version and build info - set via ldflags
var (
	Version   = "dev"
	GitCommit = "unknown"
)

const (
	AppName     = "Gala"
	Description = "A high-performance command-line tool for analyzing git repository contributions by counting lines authored by different contributors."
)

// OutputFormat represents different output formats
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"
	FormatPlain OutputFormat = "plain"
)

// SortBy represents different sorting options
type SortBy string

const (
	SortByLines SortBy = "lines"
	SortByName  SortBy = "name"
	SortByFiles SortBy = "files"
)

// Config holds application configuration
type Config struct {
	Directory     string
	Username      string
	Concurrency   int
	OutputFormat  OutputFormat
	SortBy        SortBy
	MinLines      int
	MaxResults    int
	IncludeEmoji  bool
	Quiet         bool
	Verbose       bool
	NoProgress    bool
	ExcludeAuthor []string
	IncludeAuthor []string
	DateSince     string
	DateUntil     string
	ExtraPatterns []string
	ConfigFile    string
}

// AuthorStats represents statistics for an author
type AuthorStats struct {
	Name        string  `json:"name"`
	LineCount   int     `json:"line_count"`
	FileCount   int     `json:"file_count"`
	FirstCommit string  `json:"first_commit,omitempty"`
	LastCommit  string  `json:"last_commit,omitempty"`
	Percentage  float64 `json:"percentage"`
}

// FileContribution represents a file contribution by a user
type FileContribution struct {
	Path      string `json:"path"`
	LineCount int    `json:"line_count"`
}

// AnalysisResult holds the results of git analysis
type AnalysisResult struct {
	Authors           []AuthorStats      `json:"authors"`
	UserContributions []FileContribution `json:"user_contributions,omitempty"`
	TotalLines        int                `json:"total_lines"`
	FilesProcessed    int                `json:"files_processed"`
	TotalFiles        int                `json:"total_files"`
	ProcessingTime    time.Duration      `json:"processing_time"`
	Repository        string             `json:"repository"`
	GeneratedAt       time.Time          `json:"generated_at"`
}

// Styles for consistent UI
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	primaryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14"))
)

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
		"*-lock.*", "*.lock", "Cargo.lock", "yarn.lock", "package-lock.json", "poetry.lock",
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
		"*.swp", "*.swo", "*~", "*.tmp",
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
	info, err := os.Stat(ga.config.Directory)
	if err != nil {
		return fmt.Errorf("directory %q does not exist", ga.config.Directory)
	}

	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", ga.config.Directory)
	}

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
		return nil // .gitignore doesn't exist, that's okay
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	patterns := make([]string, 0)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "!") {
			continue
		}

		pattern := line
		if strings.HasSuffix(pattern, "/") {
			pattern = strings.TrimSuffix(pattern, "/")
		}

		patterns = append(patterns, pattern)
	}

	ga.gitignoreGlobs = patterns
	if len(patterns) > 0 && ga.config.Verbose {
		ga.logInfo("Loaded %d patterns from .gitignore", len(patterns))
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

	// Check extra patterns from config
	for _, pattern := range ga.config.ExtraPatterns {
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
			return nil
		}

		if info.IsDir() {
			dirName := filepath.Base(path)
			skipDirs := []string{
				".git", "node_modules", "vendor", ".cache", "__pycache__",
				".vscode", ".idea", ".vs", "dist", "build", ".next", ".nuxt",
			}
			if slices.Contains(skipDirs, dirName) {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(ga.config.Directory, path)
		if err != nil {
			return nil
		}

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
	relPath, err := filepath.Rel(ga.config.Directory, filePath)
	if err != nil {
		return BlameResult{FilePath: filePath, Error: err}
	}

	args := []string{"blame", "-M", "-C", "-w", "--line-porcelain"}

	// Add date filtering if specified
	if ga.config.DateSince != "" {
		args = append(args, "--since="+ga.config.DateSince)
	}
	if ga.config.DateUntil != "" {
		args = append(args, "--until="+ga.config.DateUntil)
	}

	args = append(args, relPath)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = ga.config.Directory

	output, err := cmd.Output()
	if err != nil {
		return BlameResult{FilePath: filePath, Error: err}
	}

	authors := make([]string, 0)
	lines := strings.SplitSeq(string(output), "\n")

	for line := range lines {
		if strings.HasPrefix(line, "author ") {
			author := strings.TrimPrefix(line, "author ")
			if author != "" && !ga.shouldExcludeAuthor(author) {
				authors = append(authors, author)
			}
		}
	}

	return BlameResult{FilePath: filePath, Authors: authors}
}

// shouldExcludeAuthor checks if an author should be excluded
func (ga *GitAnalyzer) shouldExcludeAuthor(author string) bool {
	// Check exclude list
	for _, excluded := range ga.config.ExcludeAuthor {
		if strings.EqualFold(author, excluded) {
			return true
		}
	}

	// Check include list (if specified, only include listed authors)
	if len(ga.config.IncludeAuthor) > 0 {
		included := false
		for _, includedAuthor := range ga.config.IncludeAuthor {
			if strings.EqualFold(author, includedAuthor) {
				included = true
				break
			}
		}
		return !included
	}

	return false
}

// processFiles processes files concurrently and returns analysis results
func (ga *GitAnalyzer) processFiles(ctx context.Context, files []string) (*AnalysisResult, error) {
	startTime := time.Now()

	concurrency := ga.config.Concurrency
	if concurrency <= 0 {
		concurrency = runtime.NumCPU() * 2
	}

	var bar *progressbar.ProgressBar
	if !ga.config.NoProgress && !ga.config.Quiet {
		bar = progressbar.NewOptions(len(files),
			progressbar.OptionSetDescription("Processing files"),
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
	}

	resultsChan := make(chan BlameResult, len(files))
	g, ctx := errgroup.WithContext(ctx)
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
					if bar != nil {
						bar.Add(1)
					}
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
	authorFiles := make(map[string]map[string]bool)
	userContributions := make(map[string]int)
	totalLines := 0
	filesProcessed := 0

	for result := range resultsChan {
		if result.Error != nil {
			if ga.config.Verbose {
				ga.logWarn("Error processing %s: %v", result.FilePath, result.Error)
			}
			continue
		}

		filesProcessed++

		for _, author := range result.Authors {
			if author != "" {
				authorCounts[author]++
				totalLines++

				// Track files per author
				if authorFiles[author] == nil {
					authorFiles[author] = make(map[string]bool)
				}
				authorFiles[author][result.FilePath] = true

				// If filtering for specific user, track per-file contributions
				if ga.config.Username != "" && author == ga.config.Username {
					relPath, _ := filepath.Rel(ga.config.Directory, result.FilePath)
					userContributions[relPath]++
				}
			}
		}
	}

	if bar != nil {
		bar.Finish()
		fmt.Println()
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Convert to sorted slices
	authors := make([]AuthorStats, 0, len(authorCounts))
	for name, count := range authorCounts {
		if count >= ga.config.MinLines {
			fileCount := len(authorFiles[name])
			percentage := float64(count) / float64(totalLines) * 100
			authors = append(authors, AuthorStats{
				Name:       name,
				LineCount:  count,
				FileCount:  fileCount,
				Percentage: percentage,
			})
		}
	}

	// Sort authors
	ga.sortAuthors(authors)

	// Limit results if specified
	if ga.config.MaxResults > 0 && len(authors) > ga.config.MaxResults {
		authors = authors[:ga.config.MaxResults]
	}

	// Convert user contributions to sorted slice
	contributions := make([]FileContribution, 0, len(userContributions))
	for path, count := range userContributions {
		contributions = append(contributions, FileContribution{Path: path, LineCount: count})
	}

	sort.Slice(contributions, func(i, j int) bool {
		return contributions[i].LineCount > contributions[j].LineCount
	})

	// Limit contributions if specified
	if ga.config.MaxResults > 0 && len(contributions) > ga.config.MaxResults {
		contributions = contributions[:ga.config.MaxResults]
	}

	return &AnalysisResult{
		Authors:           authors,
		UserContributions: contributions,
		TotalLines:        totalLines,
		FilesProcessed:    filesProcessed,
		TotalFiles:        len(files),
		ProcessingTime:    time.Since(startTime),
		Repository:        ga.config.Directory,
		GeneratedAt:       time.Now(),
	}, nil
}

// sortAuthors sorts authors based on the configured sort option
func (ga *GitAnalyzer) sortAuthors(authors []AuthorStats) {
	switch ga.config.SortBy {
	case SortByLines:
		sort.Slice(authors, func(i, j int) bool {
			return authors[i].LineCount > authors[j].LineCount
		})
	case SortByName:
		sort.Slice(authors, func(i, j int) bool {
			return authors[i].Name < authors[j].Name
		})
	case SortByFiles:
		sort.Slice(authors, func(i, j int) bool {
			return authors[i].FileCount > authors[j].FileCount
		})
	}
}

// displayResults displays the analysis results based on format
func (ga *GitAnalyzer) displayResults(result *AnalysisResult) error {
	switch ga.config.OutputFormat {
	case FormatJSON:
		return ga.outputJSON(result)
	case FormatCSV:
		return ga.outputCSV(result)
	case FormatPlain:
		return ga.outputPlain(result)
	default:
		return ga.outputTable(result)
	}
}

// outputJSON outputs results in JSON format
func (ga *GitAnalyzer) outputJSON(result *AnalysisResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// outputCSV outputs results in CSV format
func (ga *GitAnalyzer) outputCSV(result *AnalysisResult) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	if ga.config.Username != "" {
		// User-specific CSV
		writer.Write([]string{"File", "Lines"})
		for _, contrib := range result.UserContributions {
			writer.Write([]string{contrib.Path, strconv.Itoa(contrib.LineCount)})
		}
	} else {
		// Authors CSV
		writer.Write([]string{"Author", "Lines", "Files", "Percentage"})
		for _, author := range result.Authors {
			writer.Write([]string{
				author.Name,
				strconv.Itoa(author.LineCount),
				strconv.Itoa(author.FileCount),
				fmt.Sprintf("%.2f", author.Percentage),
			})
		}
	}

	return nil
}

// outputPlain outputs results in plain text format
func (ga *GitAnalyzer) outputPlain(result *AnalysisResult) error {
	if ga.config.Username != "" {
		fmt.Printf("User: %s\n", ga.config.Username)
		fmt.Printf("Total Lines: %s\n", formatNumber(result.getTotalUserLines()))
		fmt.Printf("Files: %d\n\n", len(result.UserContributions))

		for _, contrib := range result.UserContributions {
			fmt.Printf("%s\t%s\n", formatNumber(contrib.LineCount), contrib.Path)
		}
	} else {
		fmt.Printf("Total Lines: %s\n", formatNumber(result.TotalLines))
		fmt.Printf("Authors: %d\n", len(result.Authors))
		fmt.Printf("Files: %d\n\n", result.FilesProcessed)

		for _, author := range result.Authors {
			fmt.Printf("%s\t%s\t%s\t%.2f%%\n",
				formatNumber(author.LineCount),
				formatNumber(author.FileCount),
				author.Name,
				author.Percentage)
		}
	}

	return nil
}

// outputTable outputs results in table format
func (ga *GitAnalyzer) outputTable(result *AnalysisResult) error {
	if ga.config.Username != "" {
		return ga.displayUserResults(result)
	}
	return ga.displayAuthorResults(result)
}

// displayAuthorResults displays results for all authors
func (ga *GitAnalyzer) displayAuthorResults(result *AnalysisResult) error {
	if !ga.config.Quiet {
		fmt.Printf("\n%s\n", ga.styleHeader("Author Contributions"))
	}

	if len(result.Authors) == 0 {
		if !ga.config.Quiet {
			ga.logWarn("No authors found matching criteria")
		}
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	headers := []string{"Rank", "Lines", "Files", "Percentage", "Author"}

	if !ga.config.IncludeEmoji {
		headers[0] = "Rank"
	}

	table.Header(headers)

	for i, author := range result.Authors {
		rank := fmt.Sprintf("%d", i+1)

		if ga.config.IncludeEmoji {
			switch i {
			case 0:
				rank = "🥇"
			case 1:
				rank = "🥈"
			case 2:
				rank = "🥉"
			}
		}

		table.Append([]string{
			rank,
			formatNumber(author.LineCount),
			formatNumber(author.FileCount),
			fmt.Sprintf("%.1f%%", author.Percentage),
			author.Name,
		})
	}

	table.Render()

	if !ga.config.Quiet {
		ga.displaySummary(result)
	}

	return nil
}

// displayUserResults displays results for a specific user
func (ga *GitAnalyzer) displayUserResults(result *AnalysisResult) error {
	if !ga.config.Quiet {
		fmt.Printf("\n%s\n", ga.styleHeader(fmt.Sprintf("%s's Contributions", ga.config.Username)))
	}

	if len(result.UserContributions) == 0 {
		if !ga.config.Quiet {
			ga.logWarn("No contributions found for user %q", ga.config.Username)
		}
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Lines", "File"})

	for _, contrib := range result.UserContributions {
		table.Append([]string{
			formatNumber(contrib.LineCount),
			contrib.Path,
		})
	}

	table.Render()

	if !ga.config.Quiet {
		summaryTable := tablewriter.NewWriter(os.Stdout)
		summaryTable.Header([]string{"Metric", "Value"})

		userTotal := result.getTotalUserLines()

		summaryTable.Append([]string{"Total lines", formatNumber(userTotal)})
		summaryTable.Append([]string{"Files contributed", formatNumber(len(result.UserContributions))})
		summaryTable.Append([]string{"Processing time", result.ProcessingTime.Round(time.Millisecond).String()})

		fmt.Printf("\n%s\n", ga.styleHeader("Summary"))
		summaryTable.Render()
	}

	return nil
}

// displaySummary displays summary statistics
func (ga *GitAnalyzer) displaySummary(result *AnalysisResult) {
	summaryTable := tablewriter.NewWriter(os.Stdout)
	summaryTable.Header([]string{"Metric", "Value"})

	summaryTable.Append([]string{"Total lines analyzed", formatNumber(result.TotalLines)})
	summaryTable.Append([]string{"Unique authors", formatNumber(len(result.Authors))})
	summaryTable.Append([]string{"Files processed", formatNumber(result.FilesProcessed)})
	summaryTable.Append([]string{"Processing time", result.ProcessingTime.Round(time.Millisecond).String()})

	fmt.Printf("\n%s\n", ga.styleHeader("Summary"))
	summaryTable.Render()
}

// getTotalUserLines calculates total lines for user contributions
func (result *AnalysisResult) getTotalUserLines() int {
	total := 0
	for _, contrib := range result.UserContributions {
		total += contrib.LineCount
	}
	return total
}

// Logging methods
func (ga *GitAnalyzer) logInfo(format string, args ...any) {
	if !ga.config.Quiet {
		fmt.Printf("[INFO] "+format+"\n", args...)
	}
}

func (ga *GitAnalyzer) logWarn(format string, args ...any) {
	if !ga.config.Quiet {
		fmt.Printf("%s "+format+"\n", append([]any{warningStyle.Render("[WARN]")}, args...)...)
	}
}

func (ga *GitAnalyzer) logError(format string, args ...any) {
	fmt.Printf("%s "+format+"\n", append([]any{errorStyle.Render("[ERROR]")}, args...)...)
}

// TODO:
func (ga *GitAnalyzer) styleHeader(text string) string {
	if ga.config.IncludeEmoji {
		return headerStyle.Render("📊 " + text)
	}
	return headerStyle.Render(text)
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
	if err := ga.validateDirectory(); err != nil {
		return err
	}

	if err := ga.loadGitignorePatterns(); err != nil {
		return fmt.Errorf("failed to load .gitignore: %w", err)
	}

	if !ga.config.Quiet {
		ga.logInfo("Scanning directory: %s", ga.config.Directory)

		if ga.config.Username != "" {
			ga.logInfo("Analyzing contributions by user: %s", ga.config.Username)
		}
	}

	files, err := ga.findFiles()
	if err != nil {
		return fmt.Errorf("failed to find files: %w", err)
	}

	if !ga.config.Quiet {
		ga.logInfo("Found %s files to analyze", formatNumber(len(files)))
	}

	if len(files) == 0 {
		ga.logWarn("No files found to analyze")
		return nil
	}

	result, err := ga.processFiles(ctx, files)
	if err != nil {
		return fmt.Errorf("failed to process files: %w", err)
	}

	return ga.displayResults(result)
}

// CLI setup
func main() {
	var config Config

	rootCmd := &cobra.Command{
		Use:     "gala [directory] [username]",
		Short:   Description,
		Long:    buildLongDescription(),
		Version: fmt.Sprintf("%s (commit: %s)", Version, GitCommit),
		Args:    cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) >= 1 {
				config.Directory = args[0]
			} else {
				config.Directory = "."
			}

			if len(args) >= 2 {
				config.Username = args[1]
			}

			absPath, err := filepath.Abs(config.Directory)
			if err != nil {
				return fmt.Errorf("invalid directory path: %w", err)
			}
			config.Directory = absPath

			analyzer := NewGitAnalyzer(config)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigChan
				if !config.Quiet {
					fmt.Printf("\nReceived interrupt signal, shutting down gracefully...\n")
				}
				cancel()
			}()

			return analyzer.Run(ctx)
		},
	}

	// Output options
	rootCmd.Flags().StringVarP((*string)(&config.OutputFormat), "output", "o", "table",
		"Output format: table, json, csv, plain")
	rootCmd.Flags().StringVar((*string)(&config.SortBy), "sort", "lines",
		"Sort by: lines, name, files")
	rootCmd.Flags().IntVar(&config.MaxResults, "limit", 0,
		"Limit number of results (0 = no limit)")
	rootCmd.Flags().BoolVar(&config.IncludeEmoji, "emoji", false,
		"Include emoji in output")

	// Filtering options
	rootCmd.Flags().IntVar(&config.MinLines, "min-lines", 1,
		"Minimum lines threshold for inclusion")
	rootCmd.Flags().StringSliceVar(&config.ExcludeAuthor, "exclude-author", nil,
		"Exclude specific authors")
	rootCmd.Flags().StringSliceVar(&config.IncludeAuthor, "include-author", nil,
		"Include only specific authors")
	rootCmd.Flags().StringVar(&config.DateSince, "since", "",
		"Only count lines since date (YYYY-MM-DD)")
	rootCmd.Flags().StringVar(&config.DateUntil, "until", "",
		"Only count lines until date (YYYY-MM-DD)")
	rootCmd.Flags().StringSliceVar(&config.ExtraPatterns, "exclude-pattern", nil,
		"Additional file patterns to exclude")

	// Behavior options
	rootCmd.Flags().IntVarP(&config.Concurrency, "concurrency", "c", 0,
		"Number of concurrent processes (default: 2*CPU cores)")
	rootCmd.Flags().BoolVarP(&config.Verbose, "verbose", "v", false,
		"Enable verbose output")
	rootCmd.Flags().BoolVarP(&config.Quiet, "quiet", "q", false,
		"Suppress all output except results")
	rootCmd.Flags().BoolVar(&config.NoProgress, "no-progress", false,
		"Disable progress bar")
	rootCmd.Flags().StringVar(&config.ConfigFile, "config", "",
		"Config file path")

	// Shell completion commands
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Long: `To load completions:

Bash:
  $ source <(gala completion bash)
  
  # To load completions for each session, execute once:
  # Linux:
  $ gala completion bash > /etc/bash_completion.d/gala
  # macOS:
  $ gala completion bash > $(brew --prefix)/etc/bash_completion.d/gala

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  
  # To load completions for each session, execute once:
  $ gala completion zsh > "${fpath[1]}/_gala"
  
  # You will need to start a new shell for this setup to take effect.

Fish:
  $ gala completion fish | source
  
  # To load completions for each session, execute once:
  $ gala completion fish > ~/.config/fish/completions/gala.fish

PowerShell:
  PS> gala completion powershell | Out-String | Invoke-Expression
  
  # To load completions for every new session, run:
  PS> gala completion powershell > gala.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}

	rootCmd.AddCommand(completionCmd)

	// Setup config file support
	if config.ConfigFile != "" {
		viper.SetConfigFile(config.ConfigFile)
	} else {
		viper.SetConfigName("gala")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.config/gala")
		viper.AddConfigPath("/etc/gala")
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("GALA")

	if err := viper.ReadInConfig(); err == nil && !config.Quiet {
		fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	}

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("%s %v\n", errorStyle.Render("[ERROR]"), err)
		os.Exit(1)
	}
}

// buildLongDescription builds the long description for the command
func buildLongDescription() string {
	return fmt.Sprintf(`%s

A professional tool for analyzing git repository contributions by counting lines 
authored by different contributors. Supports multiple output formats, filtering 
options, and advanced git blame analysis.

Examples:
  # Show all authors across all files
  gala

  # Analyze specific directory  
  gala /path/to/project

  # Show specific user's contributions per file
  gala . "John Doe"

  # Export to JSON with filtering
  gala --output json --min-lines 100 --since 2024-01-01

  # CSV output sorted by file count
  gala --output csv --sort files --limit 10

  # Exclude specific authors and patterns
  gala --exclude-author bot --exclude-pattern "*.generated.go"

Configuration:
  Gala supports configuration files in YAML format. Place gala.yaml in:
  - Current directory
  - ~/.config/gala/
  - /etc/gala/

Environment variables:
  All flags can be set via environment variables with GALA_ prefix:
  GALA_OUTPUT=json gala
  GALA_MIN_LINES=50 gala`,
		primaryStyle.Render(fmt.Sprintf("%s v%s", AppName, Version)))
}
