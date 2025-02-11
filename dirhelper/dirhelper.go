package dirhelper

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"slices"
)

// FileContent represents a file with its name and content.
type FileContent struct {
	Name    string
	Content string
	Path    string
}

func (f FileContent) FullPath() string {
	return filepath.Join(f.Path, f.Name)
}

func IsCodeFile(node fs.DirEntry) bool {
	if node.IsDir() {
		return true
	}

	codeFileExtensions := []string{".go", ".py", ".js", ".java", ".c", ".cpp", ".cs", ".rb", ".php", ".html", ".css", ".yml"}
	for _, ext := range codeFileExtensions {
		if filepath.Ext(node.Name()) == ext {
			return true
		}
	}
	return false
}

func FilterNodes(node fs.DirEntry) bool {
	skipNodes := []string{".git", ".idea", "ai_prompt.txt", "ai_knowledge.yml", "vendor", ".vscode"}
	if !IsCodeFile(node) {
		return false
	}
	if slices.Contains(skipNodes, node.Name()) {
		return false
	}
	return true
}

type FilterFunc func(node fs.DirEntry) bool

// WalkDirectories traverses a directory tree and performs a custom action on each directory.
// `root` is the starting directory.
// `onDir` is a callback function that receives:
// - Directory path
// - Files in the directory as a slice of FileContent
// - Subdirectories as a slice of strings
func WalkDirectories(root string, onDir func(directory string, files []FileContent, subdirs []string) error, filterNodes FilterFunc) error {

	if filterNodes == nil {
		filterNodes = FilterNodes
	}

	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("failed to access root directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("root path is not a directory: %s", root)
	}

	// Traverse the directory tree
	return filepath.WalkDir(root, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}

		// Only process directories
		if info.IsDir() {
			if !filterNodes(info) {
				return filepath.SkipDir
			}
			files, subdirs, err := readDirectoryContents(path, filterNodes)
			if err != nil {
				return fmt.Errorf("error reading directory contents for %s: %w", path, err)
			}
			return onDir(path, files, subdirs)
		}
		return nil
	})
}

// readDirectoryContents reads the contents of a directory and returns:
// - A slice of FileContent for all files in the directory
// - A slice of strings for all subdirectories
func readDirectoryContents(dir string, filterNodes FilterFunc) ([]FileContent, []string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading directory %s: %w", dir, err)
	}

	var files []FileContent
	var subdirs []string

	for _, entry := range entries {
		if !filterNodes(entry) {
			continue
		}
		if entry.IsDir() {
			subdirs = append(subdirs, entry.Name())
		} else {
			// Read file contents
			fullPath := filepath.Join(dir, entry.Name())
			content, err := ioutil.ReadFile(fullPath)
			if err != nil {
				return nil, nil, fmt.Errorf("error reading file %s: %w", fullPath, err)
			}
			files = append(files, FileContent{
				Name:    entry.Name(),
				Content: string(content),
				Path:    dir,
			})
		}
	}

	return files, subdirs, nil
}
