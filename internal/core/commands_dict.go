package core

import (
	"errors"
	"jedis/internal/constant"
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
