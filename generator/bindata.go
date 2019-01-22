// Code generated by "esc -o bindata.go -pkg generator -private templates/"; DO NOT EDIT.

package generator

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
)

type _escLocalFS struct{}

var _escLocal _escLocalFS

type _escStaticFS struct{}

var _escStatic _escStaticFS

type _escDirectory struct {
	fs   http.FileSystem
	name string
}

type _escFile struct {
	compressed string
	size       int64
	modtime    int64
	local      string
	isDir      bool

	once sync.Once
	data []byte
	name string
}

func (_escLocalFS) Open(name string) (http.File, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	return os.Open(f.local)
}

func (_escStaticFS) prepare(name string) (*_escFile, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	var err error
	f.once.Do(func() {
		f.name = path.Base(name)
		if f.size == 0 {
			return
		}
		var gr *gzip.Reader
		b64 := base64.NewDecoder(base64.StdEncoding, bytes.NewBufferString(f.compressed))
		gr, err = gzip.NewReader(b64)
		if err != nil {
			return
		}
		f.data, err = ioutil.ReadAll(gr)
	})
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (fs _escStaticFS) Open(name string) (http.File, error) {
	f, err := fs.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.File()
}

func (dir _escDirectory) Open(name string) (http.File, error) {
	return dir.fs.Open(dir.name + name)
}

func (f *_escFile) File() (http.File, error) {
	type httpFile struct {
		*bytes.Reader
		*_escFile
	}
	return &httpFile{
		Reader:   bytes.NewReader(f.data),
		_escFile: f,
	}, nil
}

func (f *_escFile) Close() error {
	return nil
}

func (f *_escFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func (f *_escFile) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *_escFile) Name() string {
	return f.name
}

func (f *_escFile) Size() int64 {
	return f.size
}

func (f *_escFile) Mode() os.FileMode {
	return 0
}

func (f *_escFile) ModTime() time.Time {
	return time.Unix(f.modtime, 0)
}

func (f *_escFile) IsDir() bool {
	return f.isDir
}

func (f *_escFile) Sys() interface{} {
	return f
}

// _escFS returns a http.Filesystem for the embedded assets. If useLocal is true,
// the filesystem's contents are instead used.
func _escFS(useLocal bool) http.FileSystem {
	if useLocal {
		return _escLocal
	}
	return _escStatic
}

// _escDir returns a http.Filesystem for the embedded assets on a given prefix dir.
// If useLocal is true, the filesystem's contents are instead used.
func _escDir(useLocal bool, name string) http.FileSystem {
	if useLocal {
		return _escDirectory{fs: _escLocal, name: name}
	}
	return _escDirectory{fs: _escStatic, name: name}
}

// _escFSByte returns the named file from the embedded assets. If useLocal is
// true, the filesystem's contents are instead used.
func _escFSByte(useLocal bool, name string) ([]byte, error) {
	if useLocal {
		f, err := _escLocal.Open(name)
		if err != nil {
			return nil, err
		}
		b, err := ioutil.ReadAll(f)
		_ = f.Close()
		return b, err
	}
	f, err := _escStatic.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.data, nil
}

// _escFSMustByte is the same as _escFSByte, but panics if name is not present.
func _escFSMustByte(useLocal bool, name string) []byte {
	b, err := _escFSByte(useLocal, name)
	if err != nil {
		panic(err)
	}
	return b
}

// _escFSString is the string version of _escFSByte.
func _escFSString(useLocal bool, name string) (string, error) {
	b, err := _escFSByte(useLocal, name)
	return string(b), err
}

// _escFSMustString is the string version of _escFSMustByte.
func _escFSMustString(useLocal bool, name string) string {
	return string(_escFSMustByte(useLocal, name))
}

var _escData = map[string]*_escFile{

	"/templates/context.tgo": {
		local:   "templates/context.tgo",
		size:    856,
		modtime: 1532581055,
		compressed: `
H4sIAAAAAAAC/8ySQW6zMBSE95xiFOX/BSjlAEjZULrNpvQA2H5BSNSk+KEQIe5e2QRIUBupXZUVfp43
mvlkvpwIafJca6aOYbhpJaP3ACAsatLRfOm5Yd+jyXVB2HIuKkK8xzbK7K/BMEySrRKGOLPm8R6nptR8
xOafSZNX4s11N3rRXPLldm01R3jvtShJK3saPO/YaokDneecvirziiQf8neyhUpd7KAEQvNRRWkSIFz6
jkWVkDamq/udk7UInLohbhuN/7NsNLHfPIqt526eP4L2dC31G3Lr3RW+GJrOa4K+EjK4i2ZZTk4LUqu7
QRXgzVDW+dyNILPuC5CyqrXLrYRcHk40bf6En7P6owRdtscMPwMAAP//m3s0c1gDAAA=
`,
	},

	"/templates/root.tgo": {
		local:   "templates/root.tgo",
		size:    510,
		modtime: 1530498695,
		compressed: `
H4sIAAAAAAAC/4SPzY6bMBSF936KI5TFzKL2fqpZNamK1PxIoQ9gzA22AjayL2kjxLtXhqQ/q9ldzv24
57NS+BIaQkueomZqUN9hmYf0plTr2I61NKFXV907Y10zqjaQ/4ztEYdjhd22rFB9K8/4Wn7fSaEUfiRC
uICtS0hhjIZgcoNLaMONol87NPZlhc4Z8onAVjOM9qgJlzD6Bs7nY2wJF9cRumAWu+A/tkOkISTHId6l
EIM2V90SpgkbeVo/DronzLMQrh9CZLwIAJmI2reEzSN+e8dGlsuc8Gmen9RjLx9nUPyTnTRbzHPxZMk3
y6+vQtx0RE+sz8ZSr/EOTz9fsrDc/0lfhfirwbruaLWo8piy8zSBqR86zYRiISS3oXjSK5Fb8/v+g03w
TL94xWXe/w4AAP//BUC9z/4BAAA=
`,
	},

	"/templates/table.tgo": {
		local:   "templates/table.tgo",
		size:    14801,
		modtime: 1548121434,
		compressed: `
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
`,
	},

	"/": {
		isDir: true,
		local: "",
	},

	"/templates": {
		isDir: true,
		local: "templates",
	},
}
