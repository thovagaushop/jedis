package core

import (
	"bytes"
	"errors"
	"fmt"
	"jedis/internal/constant"
	"strings"
)

const CRLF = "\r\n"

// +PING\r\n => PING, 5
func readSimpleString(data []byte) (string, int, error) {
	pos := 1

	for data[pos] != '\r' {
		pos++
	}

	return string(data[1:pos]), pos + 2, nil
}

// :123\r\n => 123
func readInt64(data []byte) (int64, int, error) {
	pos := 1
	var res int64 = 0
	for data[pos] != '\r' {
		res = res*10 + int64(data[pos]-'0')
		pos++
	}

	return res, pos + 2, nil
}

// -something\r\n
func readError(data []byte) (string, int, error) {
	return readSimpleString(data)
}

// $5\r\nhello\r\n => 5, 4
// 5 means len of string "hello"
// 4 means the position begin data (string "hello")
func readLen(data []byte) (int, int, error) {
	res, pos, _ := readInt64(data)
	return int(res), pos, nil
}

// $5\r\nhello\r\n
func readBulkString(data []byte) (string, int, error) {
	length, pos, _ := readLen(data)
	return string(data[pos : pos+length]), pos + length + 2, nil
}

// *3\r\n:1\r\n:2\r\n:3\r\n => ["1", "2", "3"]
func readArray(data []byte) ([]interface{}, int, error) {
	length, lastPos, _ := readLen(data)

	var array = make([]interface{}, length)

	for i := range array {
		value, pos, err := DecodeOne(data[lastPos:])
		if err != nil {
			return nil, 0, err
		}
		array[i] = value
		lastPos += pos
	}

	return array, lastPos, nil
}

func DecodeOne(data []byte) (interface{}, int, error) {
	if len(data) == 0 {
		return nil, 0, errors.New("no data")
	}

	switch data[0] {
	case '+':
		return readSimpleString(data)
	case '-':
		return readError(data)
	case ':':
		return readInt64(data)
	case '$':
		return readBulkString(data)
	case '*':
		return readArray(data)
	}
	return nil, 0, nil
}

func Decode(data []byte) (interface{}, error) {
	value, _, err := DecodeOne(data)
	return value, err
}

// Encode
func encodeSimpleString(s string) []byte {
	return []byte(fmt.Sprintf("+%d\r\n%s", len(s), s))
}

func encodeStringArray(strings []string) []byte {
	var b []byte
	buf := bytes.NewBuffer(b)

	for _, s := range strings {
		buf.Write(encodeSimpleString(s))
	}

	return []byte(fmt.Sprintf("*%d\r\n%s", len(strings), buf.Bytes()))
}

func Encode(value interface{}, isSimpleString bool) []byte {
	switch v := value.(type) {
	case string:
		if isSimpleString {
			return []byte(fmt.Sprintf("+%s%s", v, CRLF))
		}
		return []byte(fmt.Sprintf("$%d%s%s%s", len(v), CRLF, v, CRLF))
	case int64, int32, int16, int8, int:
		return []byte(fmt.Sprintf(":%d\r\n", v))
	case error:
		return []byte(fmt.Sprintf("-%s\r\n", v))
	case []string:
		return encodeStringArray(value.([]string))
	case [][]string:
		var b []byte
		buf := bytes.NewBuffer(b)
		for _, sa := range value.([][]string) {
			buf.Write(encodeStringArray(sa))
		}
		return []byte(fmt.Sprintf("*%d\r\n%s", len(value.([][]string)), buf.Bytes()))
	case []interface{}:
		var b []byte
		buf := bytes.NewBuffer(b)
		for _, x := range value.([]interface{}) {
			buf.Write(Encode(x, false))
		}
		return []byte(fmt.Sprintf("*%d\r\n%s", len(value.([]interface{})), buf.Bytes()))
	case []int:
		var b []byte
		buf := bytes.NewBuffer(b)
		for _, n := range value.([]int) {
			buf.Write([]byte(fmt.Sprintf("%d|", n)))
		}
		return []byte(fmt.Sprintf("@%s", buf.Bytes()))
	default:
		return []byte(constant.RESP_OK)
	}
}

func ParseCmd(data []byte) (*JedisCmd, int, error) {
	value, pos, err := DecodeOne(data)

	if err != nil {
		return nil, 0, err
	}

	array, ok := value.([]interface{})

	if !ok {
		return nil, 0, errors.New("invalid")
	}

	tokens := make([]string, len(array))

	for i, v := range array {
		tokens[i] = v.(string)
	}
	var key *string
	if len(tokens) > 1 {
		key = &tokens[1]
	}

	return &JedisCmd{
		Cmd:  strings.ToUpper(tokens[0]),
		Key:  key,
		Args: tokens[1:],
	}, pos, nil
}
