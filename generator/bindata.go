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
		size:    532,
		modtime: 1529737446,
		compressed: `
H4sIAAAAAAAC/5yQwW7CMBBE7/mKEaIViWg+IBKXNL1yKT8Q20sUKV2ovRFFUf692gCmRGoP9ckezz7t
jJyPhKp8PbDQlyCI761gSAAgaw7EefxMJnEY4GtuCEupTUcoNljmO70GjOPNsnQmkOwUXmxw9C3LHoun
UJXvJIvrbP7G0sr559hMR/bIujuJnb7GJNn3bLGlU9xz5dq6Iyvb+oM0UMvNGs4gC59dXpUpsnveS1Bn
rK45xf2NpIh0cnuS3jOeo+0C0ROlQpnrqP9V2ss11H+am8/O6ivAdJo3uHLGpg+raZc3klb6HQAA//9D
kKRFFAIAAA==
`,
	},

	"/templates/root.tgo": {
		local:   "templates/root.tgo",
		size:    510,
		modtime: 1529737446,
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
		size:    11324,
		modtime: 1529887380,
		compressed: `
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
AAD//1K9YHI8LAAA
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