# Gala - Git Author Line Analyzer

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Nix Flake](https://img.shields.io/badge/Nix-Flake-blue.svg)](https://github.com/doprz/gala)
[![Go Report Card](https://goreportcard.com/badge/github.com/doprz/gala)](https://goreportcard.com/report/github.com/doprz/gala)

A professional, high-performance command-line tool for analyzing git repository contributions by counting lines authored by different contributors. Built with Go for speed and reliability.

## Features

### 🚀 **Performance & Concurrency**

- **High-performance**: Concurrent processing with configurable worker pools
- **Memory efficient**: Streams git blame output instead of loading everything into memory
- **Smart defaults**: Automatically optimizes concurrency based on CPU cores
- **Progress tracking**: Real-time progress bars for long-running analyses

### 🎨 **Professional Output**

- **Multiple formats**: Table, JSON, CSV, and plain text output
- **Clean styling**: Professional tables with optional emoji support
- **Flexible sorting**: Sort by lines, name, or file count
- **Comprehensive stats**: Line counts, file counts, percentages, and more

### 🔍 **Advanced Analysis**

- **Dual modes**: General analysis (all authors) or user-specific (per-file contributions)
- **Smart filtering**: Excludes binary files, dependencies, and respects `.gitignore`
- **Date filtering**: Analyze contributions within specific date ranges
- **Author filtering**: Include/exclude specific authors from analysis
- **Pattern exclusion**: Custom file pattern exclusions

### 🛠 **Developer Experience**

- **Shell completions**: Bash, Zsh, Fish, and PowerShell support
- **Configuration files**: YAML configuration with multiple search paths
- **Environment variables**: All options configurable via environment
- **Verbose logging**: Detailed output for debugging and monitoring

## Installation

### Via Nix Flakes (Recommended)

Gala provides first-class Nix support with automatic shell completion installation.

#### **Direct Run (No Installation)**

```bash
# Run directly from GitHub without installing
nix run github:doprz/gala

# Run with arguments
nix run github:doprz/gala -- --help
nix run github:doprz/gala -- --output json --limit 10
```

#### **Install to Profile**

```bash
# Install to current profile
nix profile install github:doprz/gala

# Install specific version
nix profile install github:doprz/gala/v1.0.0
```

#### **Development Shell**

```bash
# Enter development environment with all tools
nix develop github:doprz/gala

# Or clone and develop locally
git clone https://github.com/doprz/gala
cd gala
nix develop  # Includes Go, golangci-lint, make, etc.
```

### NixOS Integration

Add to your NixOS configuration:

```nix
# configuration.nix or flake.nix
{
  inputs.gala.url = "github:doprz/gala";

  outputs = { self, nixpkgs, gala, ... }: {
    nixosConfigurations.myhost = nixpkgs.lib.nixosSystem {
      modules = [
        gala.nixosModules.default
        {
          programs.gala = {
            enable = true;
            settings = {
              output = "table";
              emoji = true;
              concurrency = 8;
              min-lines = 10;
              exclude-author = [ "bot" "automated" ];
            };
          };
        }
      ];
    };
  };
}
```

### Home Manager Integration

For user-level installation with configuration:

```nix
# home.nix or flake.nix
{
  inputs.gala.url = "github:doprz/gala";

  outputs = { self, home-manager, gala, ... }: {
    homeConfigurations.myuser = home-manager.lib.homeManagerConfiguration {
      modules = [
        gala.homeManagerModules.default
        {
          programs.gala = {
            enable = true;
            settings = {
              output = "table";
              emoji = true;
              sort = "lines";
              limit = 20;
              exclude-pattern = [
                "*.generated.*"
                "vendor/*"
                "node_modules/*"
              ];
            };
          };
        }
      ];
    };
  };
}
```

This automatically:

- Installs the `gala` binary
- Generates shell completions for Bash, Zsh, and Fish
- Creates `~/.config/gala/gala.yaml` with your settings

### Traditional Nix

For non-flake Nix systems:

```bash
# Install from nixpkgs (when available)
nix-env -iA nixpkgs.gala

# Or build from source
git clone https://github.com/doprz/gala
cd gala
nix-build
./result/bin/gala
```

### Via Go

```bash
# Install latest version
go install github.com/doprz/gala@latest

# Or build from source
git clone https://github.com/doprz/gala
cd gala
make install
```

### Via Docker

```bash
# Run from Docker Hub
docker run --rm -v $(pwd):/workspace ghcr.io/doprz/gala

# Build locally
docker build -t gala .
docker run --rm -v $(pwd):/workspace gala
```

### Binary Releases

Download pre-built binaries from the [releases page](https://github.com/doprz/gala/releases).

## Quick Start

### Nix Users

```bash
# Instant analysis - no installation needed
nix run github:doprz/gala

# Analyze specific repository
nix run github:doprz/gala -- /path/to/repository

# Show contributions by specific user
nix run github:doprz/gala -- . "John Doe"

# Professional output formats
nix run github:doprz/gala -- --output json --min-lines 100
```

### General Usage

```bash
# Analyze current directory
gala

# Analyze specific repository
gala /path/to/repository

# Show contributions by specific user
gala . "John Doe"

# Professional output formats
gala --output json --min-lines 100
gala --output csv --sort files --limit 10
```

## Usage

### Basic Commands

```bash
# Show all authors ranked by line contributions
gala

# Analyze specific directory
gala /path/to/project

# Show user-specific contributions per file
gala . "John Doe"

# Show help
gala --help

# Show version
gala --version
```

### Output Formats

```bash
# Table format (default) - professional tables
gala --output table

# JSON format - structured data for processing
gala --output json

# CSV format - spreadsheet compatible
gala --output csv

# Plain text - simple, parseable output
gala --output plain
```

### Filtering & Sorting

```bash
# Sort by different criteria
gala --sort lines     # By line count (default)
gala --sort name      # Alphabetically by name
gala --sort files     # By number of files contributed to

# Filter results
gala --min-lines 50               # Minimum 50 lines
gala --limit 10                   # Top 10 results only
gala --since 2024-01-01          # Since specific date
gala --until 2024-12-31          # Until specific date

# Author filtering
gala --exclude-author bot                    # Exclude bots
gala --include-author "Alice,Bob,Charlie"    # Only specific authors

# Pattern exclusion
gala --exclude-pattern "*.generated.go"     # Exclude generated files
gala --exclude-pattern "vendor/*,dist/*"    # Multiple patterns
```

### Advanced Options

```bash
# Performance tuning
gala --concurrency 16    # Use 16 worker threads
gala --no-progress       # Disable progress bar

# Output control
gala --quiet             # Minimal output
gala --verbose           # Detailed logging
gala --emoji             # Include emoji in output

# Configuration
gala --config /path/to/config.yaml    # Custom config file
```

## Configuration

Gala supports YAML configuration files for persistent settings. Configuration files are searched in:

1. `--config` flag path
2. `./gala.yaml` (current directory)
3. `~/.config/gala/gala.yaml` (user config)
4. `/etc/gala/gala.yaml` (system config)

### Example Configuration

```yaml
# ~/.config/gala/gala.yaml
output: "table"
sort: "lines"
emoji: true
concurrency: 8
min-lines: 10
exclude-author:
  - "bot"
  - "automated"
exclude-pattern:
  - "*.generated.*"
  - "vendor/*"
  - "node_modules/*"
```

### Environment Variables

All options can be set via environment variables with `GALA_` prefix:

```bash
export GALA_OUTPUT=json
export GALA_MIN_LINES=50
export GALA_EMOJI=true
gala
```

## Shell Completions

### Automatic Installation (Nix)

When using Nix, shell completions are automatically installed for Bash, Zsh, and Fish. No additional setup required!

### Manual Installation

For other installation methods, generate completions:

```bash
# Bash
gala completion bash > /etc/bash_completion.d/gala

# Zsh
gala completion zsh > "${fpath[1]}/_gala"

# Fish
gala completion fish > ~/.config/fish/completions/gala.fish

# PowerShell
gala completion powershell > gala.ps1
```

## How It Works

1. **File Discovery**: Recursively finds all files in a valid git repository with intelligent filtering
2. **Pattern Exclusion**: Skips binary files, dependencies, and `.gitignore` patterns
3. **Concurrent Processing**: Uses worker pools to run `git blame` on multiple files
4. **Data Aggregation**: Counts lines per author and generates comprehensive statistics
5. **Professional Output**: Presents results in clean, formatted tables or structured data

## Excluded File Types

Gala automatically excludes common non-source files:

- **Binary files**: Images, videos, executables, archives
- **Lock files**: `package-lock.json`, `yarn.lock`, `Cargo.lock`, etc.
- **Dependencies**: `node_modules/`, `vendor/`, cache directories
- **Generated files**: Compiled objects, minified files
- **System files**: OS-specific and IDE configuration files
- **Everything in `.gitignore`**

Additional patterns can be excluded via `--exclude-pattern` or configuration.

## Performance

Gala is optimized for performance across repository sizes.

Performance factors:

- **Concurrent processing**: Uses multiple CPU cores effectively
- **Memory efficiency**: Streams data instead of loading everything
- **Smart filtering**: Reduces I/O by excluding irrelevant files early
- **Optimized git usage**: Uses efficient `git blame` options

## Development

### Prerequisites

#### With Nix (Recommended)

```bash
# Everything included in development shell
nix develop github:doprz/gala
# Includes: Go 1.24+, Git, Make, golangci-lint, and all tools
```

#### Traditional Setup

- Go 1.24+
- Git
- Make (optional)

### Building

#### With Nix

```bash
# Clone repository
git clone https://github.com/doprz/gala
cd gala

# Enter development environment
nix develop

# Build
nix build
# or
make build

# Run tests
make test

# All development tools available
make dev  # fmt + lint + test + build
```

#### Traditional Go

```bash
# Clone repository
git clone https://github.com/doprz/gala
cd gala

# Install dependencies
go mod download

# Build binary
make build

# Run tests
make test

# Install globally
make install
```

## Acknowledgments

- Inspired by the original TypeScript/Bun implementation [gala](https://github.com/Razboy20/gala/)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
