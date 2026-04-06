package fsutil

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func LinkOrCopy(src, dst string) error {
	return LinkOrCopyCtx(context.Background(), src, dst)
}

func LinkOrCopyCtx(ctx context.Context, src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}

	err := os.Link(src, dst)
	if err == nil {
		return nil
	}

	return copyFileCtx(ctx, src, dst)
}

func copyFileCtx(ctx context.Context, src, dst string) error {
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

	// Copy in chunks so cancellation can be checked
	buf := make([]byte, 256*1024) // 256KB chunks
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		n, err := sf.Read(buf)
		if n > 0 {
			if _, wErr := df.Write(buf[:n]); wErr != nil {
				return fmt.Errorf("write data: %w", wErr)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read data: %w", err)
		}
	}

	return df.Sync()
}
