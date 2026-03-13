package datastructure

import (
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

func TestDict_Overwrite(t *testing.T) {
	dict := CreateDict()
	dict.Put("key1", "value1")
	dict.Put("key1", "value2")

	val := dict.Get("key1")
	if val != "value2" {
		t.Errorf("expected value2, got %v", val)
	}
}

func TestDict_Expiration(t *testing.T) {
	dict := CreateDict()
	dict.Put("k1", "v1")
	dict.Expire("k1", 50) // 50ms

	// Vẫn còn hạn
	if dict.Get("k1") != "v1" {
		t.Error("expected v1 to be present")
	}

	// Đợi hết hạn
	time.Sleep(60 * time.Millisecond)

	if dict.Get("k1") != nil {
		t.Error("expected k1 to be expired and deleted")
	}
}

func TestDict_Ttl(t *testing.T) {
	dict := CreateDict()
	dict.Put("k1", "v1")

	// Chưa set expire
	if dict.Ttl("k1") != 0 {
		t.Errorf("expected 0 TTL, got %v", dict.Ttl("k1"))
	}

	dict.Expire("k1", 1000)
	ttl := dict.Ttl("k1")
	now := uint64(time.Now().UnixMilli())

	if ttl < now || ttl > now+1000 {
		t.Errorf("TTL %v is out of expected range", ttl)
	}
}
