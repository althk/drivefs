// Package fusehooks implements the FUSE fs interfaces for Google Drive.
package fusehooks

import (
	"context"
	"os"
	"syscall"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/althk/drivefs/driveapi"
	"google.golang.org/api/drive/v3"
)

type FS struct {
	Ctx      context.Context
	DriveSvc *drive.Service
}

func (f *FS) Root() (fs.Node, error) {
	root := driveapi.RootFolder(f.Ctx, f.DriveSvc)
	return &Dir{
		root,
	}, nil
}

var _ fs.FS = (*FS)(nil)

type Dir struct {
	driveapi.File
}

func (d *Dir) Attr(_ context.Context, attr *fuse.Attr) error {
	return mapAttr(d.File, attr)
}

func mapAttr(f driveapi.File, a *fuse.Attr) error {
	a.Size = uint64(f.Size())
	a.Mtime = time.Now()
	a.Ctime = time.Now()
	if f.IsDir() {
		a.Mode = os.ModeDir | 0500
	} else {
		a.Mode = 0400
	}
	return nil
}

var _ fs.Node = (*Dir)(nil)

var _ = fs.HandleReadDirAller(&Dir{})

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {

	files, err := d.ListFiles(ctx)
	if err != nil {
		return nil, err
	}
	var res []fuse.Dirent

	for _, f := range files {
		var e fuse.Dirent
		e.Name = f.Name()
		if f.IsDir() {
			e.Type = fuse.DT_Dir
		} else {
			e.Type = fuse.DT_File
		}
		res = append(res, e)
	}
	return res, nil
}

var _ = fs.NodeRequestLookuper(&Dir{})

func (d *Dir) Lookup(
	_ context.Context, req *fuse.LookupRequest,
	_ *fuse.LookupResponse) (fs.Node, error) {
	name := req.Name
	for _, f := range d.Files() {
		if f.Name() == name {
			if f.IsDir() {
				return &Dir{
					f,
				}, nil
			}
			return &File{
				f,
			}, nil
		}
	}
	return nil, fuse.ToErrno(syscall.ENOENT)
}

type File struct {
	file driveapi.File
}

func (f File) Attr(_ context.Context, attr *fuse.Attr) error {
	return mapAttr(f.file, attr)
}

var _ fs.Node = (*File)(nil)
