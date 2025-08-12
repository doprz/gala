# Gala - Git Author Line Analyzer

A high-performance command-line tool for analyzing git repository contributions by counting lines authored by different contributors.

## Features

- 🚀 **High Performance**: Concurrent processing with configurable worker pools
- 🎨 **Beautiful TUI**: Clean, colorful output with progress bars and formatted tables
- 📊 **Dual Analysis Modes**: 
  - General mode: Show all authors ranked by line contributions
  - User-specific mode: Show per-file contributions for a specific user
- 🔍 **Smart Filtering**: Automatically excludes binary files, dependencies, and respects `.gitignore`
- ⚡ **Git Integration**: Uses `git blame` with advanced options for accurate attribution
- 📈 **Progress Tracking**: Real-time progress bars for long-running analyses

## Acknowledgments

- Inspired by the original TypeScript/Bun implementation [gala](https://github.com/Razboy20/gala/)

## Installation

### Prerequisites

- Go 1.24 or higher
- Git (must be in PATH)

### Build from Source

```bash
# Clone the repository
git clone https://github.com/doprz/gala
cd gala

# Download dependencies
go mod download

# Build the binary
go build -o gala .

# Optional: Install globally
go install .
```

### Binary Releases

Download pre-built binaries from the [releases page](releases).

## Usage

### Basic Usage

```bash
# Analyze current directory - show all authors
gala

# Analyze specific directory
gala /path/to/repository

# Show contributions by specific user across all files
gala . "John Doe"

# Analyze user in different directory
gala /path/to/repo alice
```

### Command Line Options

```
Usage:
  gala [directory] [username] [flags]

Flags:
  -c, --concurrency int   Number of concurrent git blame processes (default: 2*CPU cores)
  -h, --help             Show help information
  -v, --verbose          Enable verbose output
```

## How It Works

1. **Directory Validation**: Ensures the target is a valid git repository
2. **File Discovery**: Recursively finds all files, applying smart filtering
3. **Pattern Exclusion**: Skips binary files, dependencies, and `.gitignore` patterns
4. **Concurrent Processing**: Uses worker pools to run `git blame` on multiple files simultaneously
5. **Data Aggregation**: Counts lines per author and generates statistics
6. **Beautiful Output**: Presents results in formatted tables with colors and progress indicators

## Excluded File Types

Gala automatically excludes common non-source files:

- **Binary files**: Images, videos, executables, archives
- **Dependencies**: `node_modules`, `vendor`, cache directories  
- **Generated files**: Compiled objects, minified files
- **System files**: OS-specific and IDE configuration files
- **Plus**: Everything in your `.gitignore`

## Performance

- **Concurrent Processing**: Utilizes multiple CPU cores with configurable concurrency
- **Memory Efficient**: Streams git blame output instead of loading everything into memory
- **Smart Filtering**: Reduces I/O by excluding irrelevant files early
- **Progress Tracking**: Shows real-time progress for long-running operations

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

