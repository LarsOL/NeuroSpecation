package dirhelper

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// testDirEntry is a helper struct to mock fs.DirEntry for testing
type testDirEntry struct {
	name  string
	isDir bool
}

func (d testDirEntry) Name() string {
	return d.name
}

func (d testDirEntry) IsDir() bool {
	return d.isDir
}

func (d testDirEntry) Type() fs.FileMode {
	if d.isDir {
		return fs.ModeDir
	}
	return 0
}

func (d testDirEntry) Info() (fs.FileInfo, error) {
	return nil, nil
}

func TestIsCodeFile(t *testing.T) {
	testCases := []struct {
		name     string
		isDir    bool
		expected bool
	}{
		{"test.go", false, true},
		{"test.py", false, true},
		{"test.js", false, true},
		{"test.java", false, true},
		{"test.c", false, true},
		{"test.cpp", false, true},
		{"test.cs", false, true},
		{"test.rb", false, true},
		{"test.php", false, true},
		{"test.html", false, true},
		{"test.css", false, true},
		{"test.yml", false, true},
		{"test.yaml", false, true},
		{"test.md", false, true},
		{"test.txt", false, false},
		{"image.png", false, false},
		{"document.pdf", false, false},
		{"somedir", true, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := testDirEntry{name: tc.name, isDir: tc.isDir}
			if got := IsCodeFile(entry); got != tc.expected {
				t.Errorf("IsCodeFile(%q) = %v, want %v", tc.name, got, tc.expected)
			}
		})
	}
}

func TestFilterNodes(t *testing.T) {
	testCases := []struct {
		name     string
		isDir    bool
		expected bool
	}{
		{".git", true, false},
		{".idea", true, false},
		{"ai_knowledge_prompt.txt", false, false},
		{"ai_knowledge.yaml", false, false},
		{"vendor", true, false},
		{".vscode", true, false},
		{"node_modules", true, false},
		{"main.go", false, true},
		{"somedir", true, true},
		{"image.png", false, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := testDirEntry{name: tc.name, isDir: tc.isDir}
			if got := FilterNodes(entry); got != tc.expected {
				t.Errorf("FilterNodes(%q) = %v, want %v", tc.name, got, tc.expected)
			}
		})
	}
}

func TestFileContent_FullPath(t *testing.T) {
	fc := FileContent{
		Name: "file.go",
		Path: "/tmp/test",
	}
	expected := "/tmp/test/file.go"
	if got := fc.FullPath(); got != expected {
		t.Errorf("FileContent.FullPath() = %q, want %q", got, expected)
	}
}

func TestWalkDirectories(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir, err := os.MkdirTemp("", "testwalk")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectories and files
	err = os.MkdirAll(filepath.Join(tmpDir, "dir1", "dir1_1"), 0755)
	if err != nil {
		t.Fatalf("Failed to create sub-dir: %v", err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, "dir1", "file1.go"), []byte("package main"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, "dir1", "dir1_1", "file1_1.go"), []byte("package main"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	err = os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755) // Should be skipped
	if err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, ".git", "config"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file in .git: %v", err)
	}

	var paths []string
	onDir := func(directory string, files []FileContent, subdirs []string) error {
		paths = append(paths, directory)
		return nil
	}

	err = WalkDirectories(tmpDir, onDir, FilterNodes)
	if err != nil {
		t.Fatalf("WalkDirectories failed: %v", err)
	}

	expectedPaths := []string{
		tmpDir,
		filepath.Join(tmpDir, "dir1"),
		filepath.Join(tmpDir, "dir1", "dir1_1"),
	}

	if len(paths) != len(expectedPaths) {
		t.Errorf("Expected %d paths, but got %d. Got paths: %v", len(expectedPaths), len(paths), paths)
	}

	for _, p := range expectedPaths {
		found := false
		for _, gotPath := range paths {
			if p == gotPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find path %q, but it was not visited", p)
		}
	}
}

func TestReadDirectoryContents(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir, err := os.MkdirTemp("", "testread")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectories and files
	err = os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	if err != nil {
		t.Fatalf("Failed to create sub-dir: %v", err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, "file1.go"), []byte("package main"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("some text"), 0644) // Should be skipped
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	files, subdirs, err := readDirectoryContents(tmpDir, FilterNodes)
	if err != nil {
		t.Fatalf("readDirectoryContents failed: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 file, but got %d", len(files))
	} else {
		if files[0].Name != "file1.go" {
			t.Errorf("Expected file 'file1.go', but got %q", files[0].Name)
		}
		if files[0].Content != "package main" {
			t.Errorf("Expected content 'package main', but got %q", files[0].Content)
		}
	}

	if len(subdirs) != 1 {
		t.Errorf("Expected 1 subdir, but got %d", len(subdirs))
	} else if subdirs[0] != "subdir" {
		t.Errorf("Expected subdir 'subdir', but got %q", subdirs[0])
	}
}
