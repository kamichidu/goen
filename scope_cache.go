package goen

import (
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

	key := sc.Meta.KeyStringFromRowKey(rowKey)
	return sc.data[key]
}

func (sc *ScopeCache) AddObject(v interface{}) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	pk, refes := sc.Meta.RowKeysOf(v)
	pkKey := sc.Meta.KeyStringFromRowKey(pk)
	sc.data[pkKey] = v

	for _, refe := range refes {
		refeKey := sc.Meta.KeyStringFromRowKey(refe)
		if pkKey == refeKey {
			continue
		}
		var slice []interface{}
		if cached, ok := sc.data[refeKey]; ok {
			slice = cached.([]interface{})
		}
		sc.data[refeKey] = append(slice, v)
	}
}

func (sc *ScopeCache) HasObject(rowKey RowKey) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	key := sc.Meta.KeyStringFromRowKey(rowKey)
	_, ok := sc.data[key]
	return ok
}

func (sc *ScopeCache) RemoveObject(v interface{}) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	pk, refes := sc.Meta.RowKeysOf(v)
	for _, rowKey := range append(refes, pk) {
		key := sc.Meta.KeyStringFromRowKey(rowKey)
		delete(sc.data, key)
	}
}
