package asts

import (
	"go/build"
	"sync"
)

type bCacheKey struct {
	Path   string
	SrcDir string
	Mode   build.ImportMode
}

var (
	bMu sync.Mutex

	bCache = map[bCacheKey]*build.Package{}
)

func bImportDir(dir string, mode build.ImportMode) (*build.Package, error) {
	return bImport(".", dir, mode)
}

func bImport(path string, srcDir string, mode build.ImportMode) (*build.Package, error) {
	bMu.Lock()
	defer bMu.Unlock()

	cacheKey := bCacheKey{path, srcDir, mode}
	if v, ok := bCache[cacheKey]; ok {
		return v, nil
	}
	bpkg, err := build.Import(path, srcDir, mode)
	if err != nil {
		return nil, err
	}
	bCache[cacheKey] = bpkg
	return bpkg, nil
}
