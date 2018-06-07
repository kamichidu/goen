package goen

import (
	"container/list"
)

type IncludeLoader interface {
	Load(later *list.List, sc *ScopeCache, records interface{}) error
}

type IncludeLoaderFunc func(*list.List, *ScopeCache, interface{}) error

func (fn IncludeLoaderFunc) Load(later *list.List, sc *ScopeCache, records interface{}) error {
	return fn(later, sc, records)
}

type IncludeLoaderList []IncludeLoader

func (list *IncludeLoaderList) Append(v ...IncludeLoader) {
	*list = append(*list, v...)
}

func (list IncludeLoaderList) Load(later *list.List, sc *ScopeCache, records interface{}) error {
	for _, loader := range list {
		if err := loader.Load(later, sc, records); err != nil {
			return err
		}
	}
	return nil
}
