package goen

import (
	"reflect"
	"sync"
)

type ScopeCache struct {
	Meta *MetaSchema

	mu sync.RWMutex

	data map[string]interface{}
}

func NewScopeCache(meta *MetaSchema) *ScopeCache {
	return &ScopeCache{
		Meta: meta,
		data: map[string]interface{}{},
	}
}

func (sc *ScopeCache) GetObject(rowKey RowKey) interface{} {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	key, err := sc.Meta.KeyStringFromRowKey(rowKey)
	if err != nil {
		panic("goen: failed to get key string: " + err.Error())
	}
	return sc.data[key]
}

func (sc *ScopeCache) AddObject(v interface{}) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	pk, refes := sc.Meta.RowKeysOf(v)
	key, err := sc.Meta.KeyStringFromRowKey(pk)
	if err != nil {
		panic("goen: failed to get key string: " + err.Error())
	}
	sc.data[key] = v

	for _, refe := range refes {
		if reflect.DeepEqual(refe, pk) {
			continue
		}

		key, err := sc.Meta.KeyStringFromRowKey(refe)
		if err != nil {
			panic("goen: failed to get key string: " + err.Error())
		}
		var slice []interface{}
		if cached, ok := sc.data[key]; ok {
			slice = cached.([]interface{})
		}
		sc.data[key] = append(slice, v)
	}
}

func (sc *ScopeCache) HasObject(rowKey RowKey) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	key, err := sc.Meta.KeyStringFromRowKey(rowKey)
	if err != nil {
		panic("goen: failed to get key string: " + err.Error())
	}
	_, ok := sc.data[key]
	return ok
}

func (sc *ScopeCache) RemoveObject(v interface{}) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	pk, refes := sc.Meta.RowKeysOf(v)
	for _, rowKey := range append(refes, pk) {
		key, err := sc.Meta.KeyStringFromRowKey(rowKey)
		if err != nil {
			panic("goen: failed to get key string: " + err.Error())
		}
		delete(sc.data, key)
	}
}
