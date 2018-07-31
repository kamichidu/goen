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

	pk := sc.Meta.PrimaryKeyOf(v)
	pkKey := sc.Meta.KeyStringFromRowKey(pk)
	sc.data[pkKey] = v

	refes := sc.Meta.ReferenceKeysOf(v)
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

	var keys []RowKey
	keys = append(keys, sc.Meta.ReferenceKeysOf(v)...)
	keys = append(keys, sc.Meta.PrimaryKeyOf(v))
	for _, rowKey := range keys {
		key := sc.Meta.KeyStringFromRowKey(rowKey)
		delete(sc.data, key)
	}
}
