package fileserver

import (
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileServer interface defines file serving operations
type FileServer interface {
	ServeFiles(w http.ResponseWriter, r *http.Request, basePath, directory string)
	ListDirectory(w http.ResponseWriter, r *http.Request, directory string)
}

// DefaultFileServer implements the FileServer interface
type DefaultFileServer struct{}

// NewFileServer creates a new file server instance
func NewFileServer() FileServer {
	return &DefaultFileServer{}
}

// ServeFiles handles file serving requests
func (fs *DefaultFileServer) ServeFiles(w http.ResponseWriter, r *http.Request, basePath, directory string) {
	// Remove the base path from the request URL to get the relative file path
	relativePath := strings.TrimPrefix(r.URL.Path, basePath)
	if relativePath == "" {
		relativePath = "/"
	}

	// Clean the path to prevent directory traversal attacks
	relativePath = filepath.Clean(relativePath)
	if strings.HasPrefix(relativePath, "..") {
		http.Error(w, "403 Forbidden", http.StatusForbidden)
		return
	}

	// Construct the full file path
	fullPath := filepath.Join(directory, relativePath)

	// Get file info
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "404 Not Found", http.StatusNotFound)
		} else if os.IsPermission(err) {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
		} else {
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// If it's a directory, try to serve index file or show directory listing
	if fileInfo.IsDir() {
		fs.handleDirectory(w, r, fullPath, basePath, relativePath)
		return
	}

	// Serve the file
	fs.serveFile(w, r, fullPath, fileInfo)
}

// handleDirectory handles directory requests
func (fs *DefaultFileServer) handleDirectory(w http.ResponseWriter, r *http.Request, fullPath, basePath, relativePath string) {
	// Try to serve index files
	indexFiles := []string{"index.html", "index.htm", "default.html"}
	for _, indexFile := range indexFiles {
		indexPath := filepath.Join(fullPath, indexFile)
		if info, err := os.Stat(indexPath); err == nil && !info.IsDir() {
			fs.serveFile(w, r, indexPath, info)
			return
		}
	}

	// No index file found, show directory listing
	fs.ListDirectory(w, r, fullPath)
}

// serveFile serves a single file
func (fs *DefaultFileServer) serveFile(w http.ResponseWriter, r *http.Request, filePath string, fileInfo os.FileInfo) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsPermission(err) {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
		} else {
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		}
		return
	}
	defer file.Close()

	// Set content type based on file extension
	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)

	// Set other headers
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))

	// Handle range requests for large files
	http.ServeContent(w, r, fileInfo.Name(), fileInfo.ModTime(), file)
}

// ListDirectory generates and serves a directory listing
func (fs *DefaultFileServer) ListDirectory(w http.ResponseWriter, r *http.Request, directory string) {
	// Read directory contents
	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsPermission(err) {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
		} else {
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Convert to FileInfo and sort
	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't stat
		}

		files = append(files, FileInfo{
			Name:    entry.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   entry.IsDir(),
		})
	}

	// Sort files: directories first, then by name
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir // Directories first
		}
		return files[i].Name < files[j].Name
	})

	// Generate HTML response
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	data := DirectoryListing{
		Path:  r.URL.Path,
		Files: files,
	}

	if err := directoryTemplate.Execute(w, data); err != nil {
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
	}
}

// FileInfo represents file information for directory listings
type FileInfo struct {
	Name    string
	Size    int64
	ModTime time.Time
	IsDir   bool
}

// FormatSize returns a human-readable file size
func (fi FileInfo) FormatSize() string {
	if fi.IsDir {
		return "-"
	}
	
	const unit = 1024
	if fi.Size < unit {
		return fmt.Sprintf("%d B", fi.Size)
	}
	
	div, exp := int64(unit), 0
	for n := fi.Size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	return fmt.Sprintf("%.1f %cB", float64(fi.Size)/float64(div), "KMGTPE"[exp])
}

// FormatModTime returns a formatted modification time
func (fi FileInfo) FormatModTime() string {
	return fi.ModTime.Format("2006-01-02 15:04:05")
}

// DirectoryListing represents data for directory listing template
type DirectoryListing struct {
	Path  string
	Files []FileInfo
}

// directoryTemplate is the HTML template for directory listings
var directoryTemplate = template.Must(template.New("directory").Parse(`
<!DOCTYPE html>
<html>
<head>
    <title>Directory listing for {{.Path}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        h1 { color: #333; }
        table { border-collapse: collapse; width: 100%; }
        th, td { text-align: left; padding: 8px; border-bottom: 1px solid #ddd; }
        th { background-color: #f2f2f2; }
        tr:hover { background-color: #f5f5f5; }
        a { text-decoration: none; color: #0066cc; }
        a:hover { text-decoration: underline; }
        .dir { font-weight: bold; }
        .size { text-align: right; }
        .date { color: #666; }
    </style>
</head>
<body>
    <h1>Directory listing for {{.Path}}</h1>
    <table>
        <thead>
            <tr>
                <th>Name</th>
                <th>Size</th>
                <th>Last Modified</th>
            </tr>
        </thead>
        <tbody>
            {{if ne .Path "/"}}
            <tr>
                <td><a href="../" class="dir">../</a></td>
                <td class="size">-</td>
                <td class="date">-</td>
            </tr>
            {{end}}
            {{range .Files}}
            <tr>
                <td>
                    {{if .IsDir}}
                        <a href="{{.Name}}/" class="dir">{{.Name}}/</a>
                    {{else}}
                        <a href="{{.Name}}">{{.Name}}</a>
                    {{end}}
                </td>
                <td class="size">{{.FormatSize}}</td>
                <td class="date">{{.FormatModTime}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</body>
</html>
`))