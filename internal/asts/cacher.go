package asts

import (
	"go/build"
	"go/importer"
	"go/types"
	"sync"
)

var (
	SrcImporter types.Importer = &cachedImporter{
		// importer.For is deprecated since go1.12
		// but goen supports older go versions, ignore this line.
		Importer: importer.For("source", nil), // nolint: staticcheck
		cache:    map[string]*types.Package{},
	}
)

type cachedImporter struct {
	types.Importer

	mu sync.Mutex

	cache map[string]*types.Package
}

func (i *cachedImporter) Import(path string) (*types.Package, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if v, ok := i.cache[path]; ok {
		return v, nil
	}
	tpkg, err := i.Importer.Import(path)
	if err != nil {
		return nil, err
	}
	i.cache[path] = tpkg
	return tpkg, nil
}

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
