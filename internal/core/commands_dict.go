package core

import (
	"errors"
	"jedis/internal/constant"
	"strconv"
)

func cmdSet(args []string) []byte {
	if len(args) < 2 || len(args) == 3 || len(args) > 4 {
		return Encode(errors.New("(error) invalid command"), false)
	}

	key, value := args[0], args[1]
	dictStore.Put(key, value)
	return []byte(constant.RESP_OK)
}

func cmdGet(args []string) []byte {
	if len(args) <= 0 || len(args) > 1 {
		return Encode(errors.New("(error) invalid command"), false)
	}

	return Encode(dictStore.Get(args[0]), false)
}

func cmdExpire(args []string) []byte {
	if len(args) != 2 {
		return Encode(errors.New("(error) invalid command"), false)
	}

	key := args[0]
	ttlMs, err := strconv.ParseInt(args[1], 10, 64)

	if err != nil {
		return Encode(errors.New("(error) invalid ttl (ms)"), false)
	}

	value := dictStore.Get(key)

	if value == nil {
		return Encode(errors.New("(error) key not exist"), false)
	}

	dictStore.Expire(key, uint64(ttlMs))
	return []byte(constant.RESP_OK)
}

func cmdTtl(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("(error) invalid command"), false)
	}

	return Encode(dictStore.Ttl(args[0]), false)
}
