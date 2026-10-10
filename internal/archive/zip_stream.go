package archive

import (
	"archive/zip"
	"errors"
	"io"
	"time"

	"warpdrop/internal/storage"
)

// ErrNoFilesToArchive returned when attempting to create an archive from an empty file list.
var ErrNoFilesToArchive = errors.New("tidak ada berkas untuk diarsipkan")

// StreamZip compresses multiple files on-the-fly directly into the provided output writer
// using a 64 KB chunk buffer, ensuring memory footprint remains flat regardless of total archive size.
func StreamZip(w io.Writer, files []storage.FileInfo, provider storage.StorageProvider) error {
	if len(files) == 0 {
		return ErrNoFilesToArchive
	}

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	buf := make([]byte, 64*1024)
	writtenCount := 0

	for _, file := range files {
		reader, info, err := provider.GetFileReader(file.Name)
		if err != nil {
			continue
		}

		modTime := info.ModTime
		if modTime.IsZero() {
			modTime = time.Now()
		}

		header := &zip.FileHeader{
			Name:     info.Name,
			Method:   zip.Deflate,
			Modified: modTime,
		}

		headerWriter, err := zipWriter.CreateHeader(header)
		if err != nil {
			reader.Close()
			return err
		}

		_, err = io.CopyBuffer(headerWriter, reader, buf)
		reader.Close()
		if err != nil {
			return err
		}
		writtenCount++
	}

	if writtenCount == 0 {
		return ErrNoFilesToArchive
	}

	return zipWriter.Close()
}
