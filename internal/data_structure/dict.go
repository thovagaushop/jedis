package datastructure

import (
	"time"
)

type Dict struct {
	dictStore       map[string]any
	dictExpireStore map[string]uint64
}

func CreateDict() *Dict {
	return &Dict{
		dictStore:       make(map[string]any),
		dictExpireStore: make(map[string]uint64),
	}
}

func (d *Dict) Put(key string, obj any) {
	d.dictStore[key] = obj
}

func (d *Dict) Get(key string) interface{} {
	value, oke := d.dictStore[key]
	if !oke {
		return nil
	}

	expireTime, hasExpire := d.dictExpireStore[key]
	if !hasExpire {
		return value
	}

	if expireTime < uint64(time.Now().UnixMilli()) {
		// Expired
		delete(d.dictStore, key)
		delete(d.dictExpireStore, key)
		return nil
	}
	return value
}

func (d *Dict) Expire(key string, ttlMs uint64) {
	d.dictExpireStore[key] = uint64(time.Now().UnixMilli()) + ttlMs
}

func (d *Dict) Ttl(key string) uint64 {
	return d.dictExpireStore[key]
}
