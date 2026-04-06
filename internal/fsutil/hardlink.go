package fsutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func LinkOrCopy(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}

	err := os.Link(src, dst)
	if err == nil {
		return nil
	}

	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer sf.Close()

	info, err := sf.Stat()
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}

	df, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	defer func() {
		df.Close()
		// Preserve modification time
		_ = os.Chtimes(dst, info.ModTime(), info.ModTime())
	}()

	if _, err := io.Copy(df, sf); err != nil {
		return fmt.Errorf("copy data: %w", err)
	}

	return df.Sync()
}
