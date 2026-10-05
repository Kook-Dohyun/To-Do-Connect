// package-archive is a developer-only ZIP writer that preserves Unix executable
// permissions even when the release is assembled on Windows.
package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: package-archive SOURCE_DIRECTORY NEW_ZIP_FILE")
		os.Exit(1)
	}
	if err := pack(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func pack(root, destination string) error {
	f, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	z := zip.NewWriter(f)
	err = fs.WalkDir(os.DirFS(root), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular file in package: %s", path)
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name, header.Method = path, zip.Deflate
		header.SetMode(0644)
		if strings.HasPrefix(path, "plugins/todo-connect/bin/") {
			header.SetMode(0755)
		}
		out, err := z.CreateHeader(header)
		if err != nil {
			return err
		}
		in, err := os.Open(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return err
		}
		_, err = io.Copy(out, in)
		return errors.Join(err, in.Close())
	})
	return errors.Join(err, z.Close(), f.Close())
}
