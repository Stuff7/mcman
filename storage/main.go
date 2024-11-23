package storage

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

var binDir = func() string {
	executablePath, err := os.Executable()
	if err != nil {
		fmt.Println("Warning: Error getting binary directory:", err)
		return ""
	}
	return filepath.Dir(executablePath)
}()

func OpenZip(path string) (*zip.ReadCloser, error) {
	return zip.OpenReader(filepath.Join(binDir, path))
}

func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(filepath.Join(binDir, path))
}

func ReadOrCreate(path string) ([]byte, error) {
	data, err := ReadFile(path)
	if err == nil {
		return data, nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	if createErr := WriteFile(path, []byte{}); createErr != nil {
		return nil, createErr
	}

	return nil, os.ErrNotExist
}

func WriteFile(path string, data []byte) error {
	if err := MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(binDir, path), data, 0666)
}

func Open(path string) (*os.File, error) {
	if err := MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	return os.Open(filepath.Join(binDir, path))
}

func OpenFile(path string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(filepath.Join(binDir, path), flag, perm)
}

func Create(path string) (*os.File, error) {
	return os.Create(filepath.Join(binDir, path))
}

func MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(filepath.Join(binDir, path), perm)
}

func Stat(path string) (fs.FileInfo, error) {
	if err := MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	return os.Stat(filepath.Join(binDir, path))
}

func ReadFileContents(path string) ([]byte, error) {
	file, err := Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

func DirChildren(dirPath string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(binDir, dirPath))
	if err != nil {
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names, nil
}

func RemoveAll(dir string) error {
	return os.RemoveAll(filepath.Join(binDir, dir))
}
