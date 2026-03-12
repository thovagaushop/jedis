package datastructure

import (
	"strconv"
	"testing"
	"time"
)

func TestDict_PutAndGet(t *testing.T) {
	dict := CreateDict()
	dict.Put("key1", "value1")

	val := dict.Get("key1")
	if val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}
}

func TestDict_Expiration(t *testing.T) {
	dict := CreateDict()
	dict.Put("key1", "value1")
	dict.Expire("key1", 100) // 100ms

	val := dict.Get("key1")
	if val != "value1" {
		t.Errorf("expected value1 before expiration, got %v", val)
	}

	time.Sleep(150 * time.Millisecond)

	val = dict.Get("key1")
	if val != nil {
		t.Errorf("expected nil after expiration, got %v", val)
	}
}

func TestDict_NonExistentKey(t *testing.T) {
	dict := CreateDict()
	val := dict.Get("nonexistent")
	if val != nil {
		t.Errorf("expected nil for nonexistent key, got %v", val)
	}
}

func TestDict_Ttl(t *testing.T) {
	dict := CreateDict()
	dict.Put("key1", "value1")
	dict.Expire("key1", 1000)

	ttl := dict.Ttl("key1")
	now := uint64(time.Now().UnixMilli())
	if ttl <= now || ttl > now+1000 {
		t.Errorf("unexpected ttl: %v, now: %v", ttl, now)
	}

	// Key with no expiration
	dict.Put("key2", "value2")
	ttl2 := dict.Ttl("key2")
	if ttl2 != 0 {
		t.Errorf("expected 0 ttl for key with no expiration, got %v", ttl2)
	}
}

func TestDict_UpdateKey(t *testing.T) {
	dict := CreateDict()
	dict.Put("key1", "value1")
	dict.Put("key1", "value2")

	val := dict.Get("key1")
	if val != "value2" {
		t.Errorf("expected value2, got %v", val)
	}
}

func TestDict_UpdateExpiration(t *testing.T) {
	dict := CreateDict()
	dict.Put("key1", "value1")
	dict.Expire("key1", 1000)

	ttl1 := dict.Ttl("key1")
	time.Sleep(10 * time.Millisecond)
	dict.Expire("key1", 2000)
	ttl2 := dict.Ttl("key1")

	if ttl2 <= ttl1 {
		t.Errorf("expected updated ttl %v to be greater than old ttl %v", ttl2, ttl1)
	}
}

func TestDict_ConcurrentAccess(t *testing.T) {
	dict := CreateDict()
	n := 100
	done := make(chan bool)

	for i := 0; i < n; i++ {
		go func(i int) {
			key := strconv.Itoa(i)
			dict.Put(key, i)
			dict.Get(key)
			done <- true
		}(i)
	}

	for i := 0; i < n; i++ {
		<-done
	}
}
