# WALL-E

> WALL-E scans your codebase and trashes your comments.

WALL-E is a CLI tool for removing comments from your codebase. Run `walle fix` to get rid of all comments that are in your uncommitted changes.

> **Note**: both `walle` and `wall-e` are valid commands.
> 
## Installation

### Homebrew (macOS/Linux)

```bash
brew tap kallepronk/walle
brew install walle
```

### From Source

```bash
git clone https://github.com/kallepronk/python-comment-remover.git
cd python-comment-remover
go build -o walle .
```

## Usage

### Scan for Comments

Find comments without removing them:

```bash
# Scan only changed files (git diff)
walle scan

# Scan all files in the current directory
walle scan -a

# Scan a specific file or directory
walle scan -p path/to/file.py

# Verbose output (shows each comment with line numbers)
walle scan -v
```

### Remove Comments

Remove comments from your codebase:

```bash
# Remove comments from changed files (with confirmation prompt)
walle fix

# Remove comments from all files
walle fix -a

# Remove comments from a specific file or directory
walle fix -p path/to/file.py

# Interactive mode - select which comments to remove
walle fix -i

# Force mode - skip confirmation prompt
walle fix -f
```

## 📋 Commands

| Command | Description |
|---------|-------------|
| `walle scan` | Find comments without deleting them |
| `walle fix` | Remove comments from files |
| `walle help` | Help about any command |

## 🎛️ Flags

### Scan Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--all` | `-a` | Scan all files in the current directory |
| `--path` | `-p` | Scan a specific file or directory |
| `--verbose` | `-v` | Show detailed output with line numbers |

### Fix Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--all` | `-a` | Fix all files in the current directory |
| `--path` | `-p` | Fix a specific file or directory |
| `--interactive` | `-i` | Interactively select which comments to remove |
| `--force` | `-f` | Skip confirmation prompt |

## 🔧 How It Works

1. **Default Behavior**: WALL-E uses git to detect only changed files, ensuring you only clean up new comments you've added
2. **Scanning**: Uses tree-sitter for accurate parsing to identify comments in your code
3. **Removal**: Safely removes selected comments while preserving your code structure

## 📝 Example

```bash
❯ walle scan
Found 11 comments in read_qr_code.py
Found 43 comments in cv2_bounding_box.py
Found 29 comments in we_chat_qr_code.py
Found 294 comments in 6 files

❯ walle scan -v
Found 11 comments in read_qr_code.py
        - Line 9: # Load the image
        - Line 15: # Use OpenCV QRCodeDetector
        ...

❯ walle fix -i
Found 294 comments in 6 files.
# Interactive TUI opens to select comments for removal

❯ walle fix -f
Found 294 comments in 6 files.
✅ Removed 11 comments from read_qr_code.py
✅ Removed 43 comments from cv2_bounding_box.py
...
🗑️  Trash compacted 294 comments total.
```

