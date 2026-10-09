package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"warpdrop/internal/security"
)

// FileInfo contains metadata for a stored file.
type FileInfo struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	HumanSize string    `json:"human_size"`
	ModTime   time.Time `json:"mod_time"`
}

// StorageProvider abstracts underlying storage mechanisms (disk, vault, cloud).
type StorageProvider interface {
	SaveStream(filename string, r io.Reader) (FileInfo, error)
	ListFiles() ([]FileInfo, error)
	GetFileReader(filename string) (io.ReadCloser, FileInfo, error)
	GetBaseDir() string
}

// DiskStorageProvider implements StorageProvider for standard filesystem storage.
type DiskStorageProvider struct {
	baseDir string
}

// NewDiskStorageProvider instantiates a DiskStorageProvider and ensures base directory exists.
func NewDiskStorageProvider(baseDir string) (*DiskStorageProvider, error) {
	absDir, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(absDir, 0755); err != nil {
		return nil, err
	}

	return &DiskStorageProvider{baseDir: absDir}, nil
}

// GetBaseDir returns absolute path of base storage directory.
func (p *DiskStorageProvider) GetBaseDir() string {
	return p.baseDir
}

// SaveStream writes an incoming reader stream directly to disk with buffered I/O.
func (p *DiskStorageProvider) SaveStream(filename string, r io.Reader) (FileInfo, error) {
	destPath, err := security.GetSafeDestinationPath(p.baseDir, filename)
	if err != nil {
		return FileInfo{}, err
	}

	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return FileInfo{}, err
	}
	defer destFile.Close()

	// 64 KB buffer for memory-efficient streaming
	buf := make([]byte, 64*1024)
	written, err := io.CopyBuffer(destFile, r, buf)
	if err != nil {
		os.Remove(destPath)
		return FileInfo{}, err
	}

	stat, err := destFile.Stat()
	if err != nil {
		return FileInfo{}, err
	}

	return FileInfo{
		Name:      filepath.Base(destPath),
		Size:      written,
		HumanSize: FormatBytes(written),
		ModTime:   stat.ModTime(),
	}, nil
}

// ListFiles returns sorted list of files stored in base directory (newest first).
func (p *DiskStorageProvider) ListFiles() ([]FileInfo, error) {
	entries, err := os.ReadDir(p.baseDir)
	if err != nil {
		return nil, err
	}

	var results []FileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		results = append(results, FileInfo{
			Name:      entry.Name(),
			Size:      info.Size(),
			HumanSize: FormatBytes(info.Size()),
			ModTime:   info.ModTime(),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ModTime.After(results[j].ModTime)
	})

	return results, nil
}

// GetFileReader opens a stored file for read streaming after sanitizing input name.
func (p *DiskStorageProvider) GetFileReader(filename string) (io.ReadCloser, FileInfo, error) {
	safeName := security.SanitizeFilename(filename)
	targetPath := filepath.Join(p.baseDir, safeName)

	stat, err := os.Stat(targetPath)
	if err != nil {
		return nil, FileInfo{}, err
	}

	if stat.IsDir() {
		return nil, FileInfo{}, os.ErrNotExist
	}

	file, err := os.Open(targetPath)
	if err != nil {
		return nil, FileInfo{}, err
	}

	return file, FileInfo{
		Name:      stat.Name(),
		Size:      stat.Size(),
		HumanSize: FormatBytes(stat.Size()),
		ModTime:   stat.ModTime(),
	}, nil
}

// FormatBytes formats byte counts into human-readable strings.
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
