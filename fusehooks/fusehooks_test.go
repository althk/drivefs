package fusehooks

import (
	"bytes"
	"context"
	"io"
	"os"
	"reflect"
	"testing"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/althk/drivefs/driveapi"
)

// mockFile implements driveapi.File interface
// for testing
type mockFile struct {
	name, mimeType, parentName, parentID, id string
	size                                     uint64
	isDir, isGoogleAppsFile                  bool
	content                                  []byte
	files                                    []driveapi.File
}

func (f *mockFile) String() string {
	return f.name
}

func (f *mockFile) IsDir() bool {
	return f.isDir
}

func (f *mockFile) IsGoogleAppsFile() bool {
	return f.isGoogleAppsFile
}

func (f *mockFile) ListFiles(ctx context.Context) ([]driveapi.File, error) {
	return f.files, nil
}

func (f *mockFile) Size() uint64 {
	return f.size
}

func (f *mockFile) Name() string {
	return f.name
}

func (f *mockFile) MimeType() string {
	return f.mimeType
}

func (f *mockFile) ID() string {
	return f.id
}

func (f *mockFile) ParentID() string {
	return f.parentID
}

func (f *mockFile) ParentName() string {
	return f.parentName
}

func (f *mockFile) Content() []byte {
	return f.content
}

func (f *mockFile) Files() []driveapi.File {
	return f.files
}

func (f *mockFile) Download(_ context.Context) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(f.content)), nil
}

var root = &mockFile{
	name:             "My Drive",
	mimeType:         driveapi.GoogleAppsMimeTypeText(driveapi.MimeTypeGoogleDriveFolder),
	parentName:       "",
	parentID:         "",
	id:               "root",
	size:             0,
	isDir:            true,
	isGoogleAppsFile: false,
	content:          []byte{},
	files: []driveapi.File{
		fileA,
		dirB,
	},
}

var fileAContent = []byte("fileA contents")
var fileA = &mockFile{
	name:             "file-a",
	mimeType:         "application/json",
	parentName:       "My Drive",
	parentID:         "id1",
	id:               "fid2",
	size:             uint64(len(fileAContent)),
	isDir:            false,
	isGoogleAppsFile: false,
	content:          fileAContent,
	files:            []driveapi.File{},
}

var dirB = &mockFile{
	name:             "dir-b",
	mimeType:         driveapi.GoogleAppsMimeTypeText(driveapi.MimeTypeGoogleDriveFolder),
	parentName:       "My Drive",
	parentID:         "id1",
	id:               "did3",
	size:             0,
	isDir:            true,
	isGoogleAppsFile: false,
	content:          []byte{},
	files:            []driveapi.File{},
}

func TestFS_Root(t *testing.T) {
	type fields struct {
		Ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		want    fs.Node
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FS{
				Ctx: tt.fields.Ctx,
			}
			got, err := f.Root()
			if (err != nil) != tt.wantErr {
				t.Errorf("FS.Root() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FS.Root() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDir_Attr(t *testing.T) {
	type fields struct {
		File driveapi.File
	}
	type args struct {
		in0  context.Context
		attr *fuse.Attr
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Dir_Attr with directory",
			fields: fields{
				File: dirB,
			},
			args: args{
				in0:  context.TODO(),
				attr: &fuse.Attr{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Dir{
				File: tt.fields.File,
			}
			if err := d.Attr(tt.args.in0, tt.args.attr); (err != nil) != tt.wantErr {
				t.Errorf("Dir.Attr() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.args.attr.Mode != (os.ModeDir | 0500) {
				t.Errorf("Dir.Attr(%s) => %q\t\t want %q",
					tt.fields.File, tt.args.attr.Mode, (os.ModeDir | 0500))
			}
		})
	}
}

func TestDir_ReadDirAll(t *testing.T) {
	type fields struct {
		File driveapi.File
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []fuse.Dirent
		wantErr bool
	}{
		{
			name: "Dir_ReadDirAll",
			fields: fields{
				File: root,
			},
			args:    args{context.TODO()},
			wantErr: false,
			want: []fuse.Dirent{
				{
					Name: fileA.Name(),
					Type: fuse.DT_File,
				},
				{
					Name: dirB.Name(),
					Type: fuse.DT_Dir,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Dir{
				File: tt.fields.File,
			}
			got, err := d.ReadDirAll(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Dir.ReadDirAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Dir.ReadDirAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDir_Lookup(t *testing.T) {
	type fields struct {
		File driveapi.File
	}
	type args struct {
		in0 context.Context
		req *fuse.LookupRequest
		in2 *fuse.LookupResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    fs.Node
		wantErr bool
	}{
		{
			name: "Dir_Lookup on a regular file",
			fields: fields{
				File: root,
			},
			args: args{
				in0: context.TODO(),
				req: &fuse.LookupRequest{
					Name: "file-a",
				},
			},
			want: &File{
				fileA,
			},
		},
		{
			name: "Dir_Lookup on a sub dir",
			fields: fields{
				File: root,
			},
			args: args{
				in0: context.TODO(),
				req: &fuse.LookupRequest{
					Name: "dir-b",
				},
			},
			want: &Dir{
				dirB,
			},
		},
		{
			name: "Dir_Lookup on a non existent file",
			fields: fields{
				File: root,
			},
			args: args{
				in0: context.TODO(),
				req: &fuse.LookupRequest{
					Name: "na",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Dir{
				File: tt.fields.File,
			}
			got, err := d.Lookup(tt.args.in0, tt.args.req, tt.args.in2)
			if (err != nil) != tt.wantErr {
				t.Errorf("Dir.Lookup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Dir.Lookup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFile_Attr(t *testing.T) {
	type fields struct {
		file driveapi.File
	}
	type args struct {
		in0  context.Context
		attr *fuse.Attr
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "File_Attr with regular file",
			fields: fields{
				file: fileA,
			},
			args: args{
				in0:  context.TODO(),
				attr: &fuse.Attr{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := File{
				file: tt.fields.file,
			}
			if err := f.Attr(tt.args.in0, tt.args.attr); (err != nil) != tt.wantErr {
				t.Errorf("File.Attr() error = %v, wantErr %v", err, tt.wantErr)
				if tt.args.attr.Mode != 0400 {
					t.Errorf("Dir.Attr(%s) => %q\t\t want %q",
						tt.fields.file, tt.args.attr.Mode, 0400)
				}
			}
		})
	}
}

func TestFile_Open(t *testing.T) {
	type fields struct {
		file driveapi.File
	}
	type args struct {
		ctx  context.Context
		req  *fuse.OpenRequest
		resp *fuse.OpenResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    fs.Handle
		wantErr bool
	}{
		{name: "File_Open",
			fields: fields{
				file: fileA,
			},
			args: args{
				ctx:  context.TODO(),
				req:  &fuse.OpenRequest{},
				resp: &fuse.OpenResponse{},
			},
			want: &FileHandle{
				io.NopCloser(bytes.NewReader(fileA.Content())),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &File{
				file: tt.fields.file,
			}
			got, err := f.Open(tt.args.ctx, tt.args.req, tt.args.resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("File.Open() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("File.Open() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileHandle_Read(t *testing.T) {
	type fields struct {
		r io.ReadCloser
	}
	type args struct {
		ctx  context.Context
		req  *fuse.ReadRequest
		resp *fuse.ReadResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "FileHandle_Read",
			fields: fields{
				r: io.NopCloser(bytes.NewReader(fileA.Content())),
			},
			args: args{
				ctx: context.TODO(),
				req: &fuse.ReadRequest{
					Size: int(fileA.Size()),
				},
				resp: &fuse.ReadResponse{},
			},
			wantErr: false,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh := &FileHandle{
				r: tt.fields.r,
			}
			if err := fh.Read(tt.args.ctx, tt.args.req, tt.args.resp); (err != nil) != tt.wantErr {
				t.Errorf("FileHandle.Read() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !bytes.Equal(tt.args.resp.Data, fileA.Content()) {
				t.Errorf("FileHandle.Read() => \n%q, want %q", tt.args.resp.Data, fileA.Content())
			}
			fh.Release(context.TODO(), &fuse.ReleaseRequest{}) // Ensure Release does not error.
		})
	}
}
