package release

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func DeterministicZip(destination, root string, files []string, timestamp time.Time) error {
	if timestamp.Year() < 1980 {
		timestamp = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	timestamp = timestamp.UTC().Truncate(2 * time.Second)
	sorted := append([]string(nil), files...)
	sort.Strings(sorted)
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	archive := zip.NewWriter(output)
	for _, relative := range sorted {
		clean := filepath.Clean(relative)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			_ = archive.Close()
			_ = output.Close()
			return os.ErrPermission
		}
		path := filepath.Join(root, clean)
		info, err := os.Stat(path)
		if err != nil {
			_ = archive.Close()
			_ = output.Close()
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			_ = archive.Close()
			_ = output.Close()
			return err
		}
		header.Name = filepath.ToSlash(clean)
		header.Method = zip.Deflate
		header.Modified = timestamp
		writer, err := archive.CreateHeader(header)
		if err != nil {
			_ = archive.Close()
			_ = output.Close()
			return err
		}
		input, err := os.Open(path)
		if err != nil {
			_ = archive.Close()
			_ = output.Close()
			return err
		}
		_, copyErr := io.Copy(writer, input)
		closeErr := input.Close()
		if copyErr != nil {
			_ = archive.Close()
			_ = output.Close()
			return copyErr
		}
		if closeErr != nil {
			_ = archive.Close()
			_ = output.Close()
			return closeErr
		}
	}
	if err := archive.Close(); err != nil {
		_ = output.Close()
		return err
	}
	return output.Close()
}

func FilesUnder(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, relative)
		return nil
	})
	sort.Strings(files)
	return files, err
}
