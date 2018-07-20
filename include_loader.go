package goen

import (
	"container/list"
	"reflect"
)

type IncludeBuffer list.List

func (l *IncludeBuffer) AddRecords(records interface{}) {
	typ := reflect.TypeOf(records)
	if typ.Kind() != reflect.Slice {
		panic("goen: AddRecords only accepts a slice of entities, not " + typ.String())
	}
	(*list.List)(l).PushBack(records)
}

type IncludeLoader interface {
	Load(later *IncludeBuffer, sc *ScopeCache, records interface{}) error
}

type IncludeLoaderFunc func(*IncludeBuffer, *ScopeCache, interface{}) error

func (fn IncludeLoaderFunc) Load(later *IncludeBuffer, sc *ScopeCache, records interface{}) error {
	return fn(later, sc, records)
}

type IncludeLoaderList []IncludeLoader

func (list *IncludeLoaderList) Append(v ...IncludeLoader) {
	*list = append(*list, v...)
}

func (list IncludeLoaderList) Load(later *IncludeBuffer, sc *ScopeCache, records interface{}) error {
	for _, loader := range list {
		if err := loader.Load(later, sc, records); err != nil {
			return err
		}
	}
	return nil
}
