package files

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"path"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
)

type TarWriter struct {
	TarW *tar.Writer
}

// NewTarWriter wraps given io.Writer into a new tar writer
func NewTarWriter(w io.Writer) (*TarWriter, error) {
	return &TarWriter{
		TarW: tar.NewWriter(w),
	}, nil
}

func (w *TarWriter) writeDir(ctx context.Context, f Directory, fpath string) error {
	if err := writeDirHeader(w.TarW, fpath); err != nil {
		return err
	}

	it := f.Entries()
	for it.Next() {
		if err := w.WriteFile(ctx, it.Node(), path.Join(fpath, it.Name())); err != nil {
			return err
		}
	}
	return it.Err()
}

func (w *TarWriter) writeFile(f File, fpath string) error {
	size, err := f.Size()
	if err != nil {
		return err
	}

	if err := writeFileHeader(w.TarW, fpath, uint64(size)); err != nil {
		return err
	}

	if _, err := io.Copy(w.TarW, f); err != nil {
		return err
	}
	w.TarW.Flush()
	return nil
}

// WriteNode adds a node to the archive.
func (w *TarWriter) WriteFile(ctx context.Context, nd Node, fpath string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "tar write file")
	defer span.Finish()
	span.SetTag("path", fpath)

	switch nd := nd.(type) {
	case *Symlink:
		span.SetTag("type", "symlink")
		return writeSymlinkHeader(w.TarW, nd.Target, fpath)
	case File:
		span.SetTag("type", "file")
		return w.writeFile(nd, fpath)
	case Directory:
		span.SetTag("type", "directory")
		return w.writeDir(ctx, nd, fpath)
	default:
		return fmt.Errorf("file type %T is not supported", nd)
	}
}

// Close closes the tar writer.
func (w *TarWriter) Close() error {
	return w.TarW.Close()
}

func writeDirHeader(w *tar.Writer, fpath string) error {
	return w.WriteHeader(&tar.Header{
		Name:     fpath,
		Typeflag: tar.TypeDir,
		Mode:     0777,
		ModTime:  time.Now(),
		// TODO: set mode, dates, etc. when added to unixFS
	})
}

func writeFileHeader(w *tar.Writer, fpath string, size uint64) error {
	return w.WriteHeader(&tar.Header{
		Name:     fpath,
		Size:     int64(size),
		Typeflag: tar.TypeReg,
		Mode:     0644,
		ModTime:  time.Now(),
		// TODO: set mode, dates, etc. when added to unixFS
	})
}

func writeSymlinkHeader(w *tar.Writer, target, fpath string) error {
	return w.WriteHeader(&tar.Header{
		Name:     fpath,
		Linkname: target,
		Mode:     0777,
		Typeflag: tar.TypeSymlink,
	})
}
