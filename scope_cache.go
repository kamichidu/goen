package goen

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"sync"
)

type Cardinality int

const (
	CardinalityNone Cardinality = iota
	CardinalityOneToMany
	CardinalityManyToOne
)

func (v Cardinality) toCacheKey(keyStr string) string {
	switch v {
	case CardinalityNone:
		return keyStr
	case CardinalityOneToMany:
		return keyStr + "#cardinality=OneToMany"
	case CardinalityManyToOne:
		return keyStr + "#cardinality=ManyToOne"
	default:
		panic(fmt.Sprintf("goen: invalid Cardinality: %v", v))
	}
}

type ScopeCache struct {
	Meta MetaSchema

	mu sync.RWMutex

	data map[string]interface{}
}

func NewScopeCache(meta MetaSchema) *ScopeCache {
	return &ScopeCache{
		Meta: meta,
		data: map[string]interface{}{},
	}
}

// GetObject gets stored object with given cardinality
func (sc *ScopeCache) GetObject(cardinality Cardinality, rowKey RowKey) interface{} {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	key := cardinality.toCacheKey(sc.Meta.KeyStringFromRowKey(rowKey))
	if v, ok := sc.data[key]; ok {
		return v
	} else {
		return nil
	}
}

func (sc *ScopeCache) AddObject(v interface{}) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	pk := sc.Meta.PrimaryKeyOf(v)
	pkKey := CardinalityNone.toCacheKey(sc.Meta.KeyStringFromRowKey(pk))
	sc.data[pkKey] = v

	var refes []RowKey
	// add v as one-to-many relation
	refes = sc.Meta.OneToManyReferenceKeysOf(v)
	for _, refe := range refes {
		refeKey := CardinalityOneToMany.toCacheKey(sc.Meta.KeyStringFromRowKey(refe))
		var slice []interface{}
		if cached, ok := sc.data[refeKey]; ok {
			slice = cached.([]interface{})
		}
		sc.data[refeKey] = append(slice, v)
	}
	// add v as many-to-one relation
	refes = sc.Meta.ManyToOneReferenceKeysOf(v)
	for _, refe := range refes {
		refeKey := CardinalityManyToOne.toCacheKey(sc.Meta.KeyStringFromRowKey(refe))
		sc.data[refeKey] = v
	}
}

// HasObject checks stored object with given cardinality is exists or not.
func (sc *ScopeCache) HasObject(cardinality Cardinality, rowKey RowKey) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	key := cardinality.toCacheKey(sc.Meta.KeyStringFromRowKey(rowKey))
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
		for _, cardinality := range []Cardinality{
			CardinalityNone,
			CardinalityOneToMany,
			CardinalityManyToOne,
		} {
			delete(sc.data, cardinality.toCacheKey(key))
		}
	}
}

// for debugging
func (sc *ScopeCache) Dump(w io.Writer) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	var keys []string
	for k := range sc.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Fprintf(w, "> --- ScopeCache begin ---\n")
	for i, key := range keys {
		val := sc.data[key]
		rv := reflect.ValueOf(val)
		for rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.Slice {
			fmt.Fprintf(w, "> %03d | %s = %#v (len=%d)\n", i, key, val, rv.Len())
		} else {
			fmt.Fprintf(w, "> %03d | %s = %T\n", i, key, val)
		}
	}
	fmt.Fprintf(w, "> --- ScopeCache end ---\n")
}
