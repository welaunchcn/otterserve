package fileserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewFileServer(t *testing.T) {
	fs := NewFileServer()
	if fs == nil {
		t.Fatal("Expected file server to be created")
	}
}

func TestFileServer_ServeFiles_File(t *testing.T) {
	fs := NewFileServer()
	
	// Create a temporary directory with test files
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test serving the file
	req := httptest.NewRequest("GET", "/static/test.txt", nil)
	rr := httptest.NewRecorder()

	fs.ServeFiles(rr, req, "/static", tempDir)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if rr.Body.String() != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, rr.Body.String())
	}

	// Check content type
	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("Expected text/plain content type, got '%s'", contentType)
	}
}

func TestFileServer_ServeFiles_FileNotFound(t *testing.T) {
	fs := NewFileServer()
	tempDir := t.TempDir()

	req := httptest.NewRequest("GET", "/static/nonexistent.txt", nil)
	rr := httptest.NewRecorder()

	fs.ServeFiles(rr, req, "/static", tempDir)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestFileServer_ServeFiles_DirectoryTraversal(t *testing.T) {
	fs := NewFileServer()
	tempDir := t.TempDir()

	// Try to access parent directory
	req := httptest.NewRequest("GET", "/static/../../../etc/passwd", nil)
	rr := httptest.NewRecorder()

	fs.ServeFiles(rr, req, "/static", tempDir)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status %d for directory traversal, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestFileServer_ServeFiles_DirectoryWithIndex(t *testing.T) {
	fs := NewFileServer()
	tempDir := t.TempDir()
	
	// Create index.html file
	indexFile := filepath.Join(tempDir, "index.html")
	indexContent := "<html><body>Index Page</body></html>"
	
	err := os.WriteFile(indexFile, []byte(indexContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create index file: %v", err)
	}

	// Request directory
	req := httptest.NewRequest("GET", "/static/", nil)
	rr := httptest.NewRecorder()

	fs.ServeFiles(rr, req, "/static", tempDir)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if rr.Body.String() != indexContent {
		t.Errorf("Expected index content, got '%s'", rr.Body.String())
	}

	// Check content type
	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected text/html content type, got '%s'", contentType)
	}
}

func TestFileServer_ServeFiles_DirectoryWithoutIndex(t *testing.T) {
	fs := NewFileServer()
	tempDir := t.TempDir()
	
	// Create some test files
	testFile1 := filepath.Join(tempDir, "file1.txt")
	testFile2 := filepath.Join(tempDir, "file2.txt")
	
	os.WriteFile(testFile1, []byte("content1"), 0644)
	os.WriteFile(testFile2, []byte("content2"), 0644)

	// Request directory (should show listing)
	req := httptest.NewRequest("GET", "/static/", nil)
	rr := httptest.NewRecorder()

	fs.ServeFiles(rr, req, "/static", tempDir)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	body := rr.Body.String()
	
	// Check that it's HTML directory listing
	if !strings.Contains(body, "<html>") {
		t.Error("Expected HTML directory listing")
	}
	
	// Check that files are listed
	if !strings.Contains(body, "file1.txt") {
		t.Error("Expected file1.txt in directory listing")
	}
	if !strings.Contains(body, "file2.txt") {
		t.Error("Expected file2.txt in directory listing")
	}
}

func TestFileServer_ListDirectory(t *testing.T) {
	fs := NewFileServer()
	tempDir := t.TempDir()
	
	// Create test files and directories
	testFile := filepath.Join(tempDir, "test.txt")
	testDir := filepath.Join(tempDir, "subdir")
	
	os.WriteFile(testFile, []byte("test content"), 0644)
	os.MkdirAll(testDir, 0755)

	req := httptest.NewRequest("GET", "/test/", nil)
	rr := httptest.NewRecorder()

	fs.ListDirectory(rr, req, tempDir)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	body := rr.Body.String()
	
	// Check content type
	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected text/html content type, got '%s'", contentType)
	}
	
	// Check directory listing content
	if !strings.Contains(body, "Directory listing for /test/") {
		t.Error("Expected directory listing title")
	}
	if !strings.Contains(body, "test.txt") {
		t.Error("Expected test.txt in listing")
	}
	if !strings.Contains(body, "subdir/") {
		t.Error("Expected subdir/ in listing")
	}
}

func TestFileServer_ListDirectory_PermissionDenied(t *testing.T) {
	fs := NewFileServer()
	
	// Try to list a directory that doesn't exist
	req := httptest.NewRequest("GET", "/test/", nil)
	rr := httptest.NewRecorder()

	fs.ListDirectory(rr, req, "/nonexistent/directory")

	// Should return 500 for read error (not permission in this case, but similar handling)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestFileInfo_FormatSize(t *testing.T) {
	tests := []struct {
		name     string
		fileInfo FileInfo
		expected string
	}{
		{
			name:     "directory",
			fileInfo: FileInfo{IsDir: true, Size: 0},
			expected: "-",
		},
		{
			name:     "small file",
			fileInfo: FileInfo{IsDir: false, Size: 512},
			expected: "512 B",
		},
		{
			name:     "kilobyte file",
			fileInfo: FileInfo{IsDir: false, Size: 1536}, // 1.5 KB
			expected: "1.5 KB",
		},
		{
			name:     "megabyte file",
			fileInfo: FileInfo{IsDir: false, Size: 2097152}, // 2 MB
			expected: "2.0 MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fileInfo.FormatSize()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestFileInfo_FormatModTime(t *testing.T) {
	testTime := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)
	fileInfo := FileInfo{ModTime: testTime}
	
	result := fileInfo.FormatModTime()
	expected := "2023-12-25 15:30:45"
	
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFileServer_ServeFile_ContentTypes(t *testing.T) {
	fs := &DefaultFileServer{}
	tempDir := t.TempDir()

	tests := []struct {
		filename    string
		content     string
		expectedCT  string
	}{
		{"test.html", "<html></html>", "text/html"},
		{"test.css", "body { color: red; }", "text/css"},
		{"test.js", "console.log('test');", "text/javascript"},
		{"test.json", `{"key": "value"}`, "application/json"},
		{"test.png", "fake png content", "image/png"},
		{"test.unknown", "unknown content", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.filename)
			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			fileInfo, err := os.Stat(filePath)
			if err != nil {
				t.Fatalf("Failed to stat test file: %v", err)
			}

			req := httptest.NewRequest("GET", "/"+tt.filename, nil)
			rr := httptest.NewRecorder()

			fs.serveFile(rr, req, filePath, fileInfo)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
			}

			contentType := rr.Header().Get("Content-Type")
			if !strings.Contains(contentType, tt.expectedCT) {
				t.Errorf("Expected content type to contain '%s', got '%s'", tt.expectedCT, contentType)
			}
		})
	}
}