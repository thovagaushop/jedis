package resp

import (
	"errors"
)

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

	print(length, pos)

	return string(data[pos : pos+length]), pos + length + 2, nil
}

// *3\r\n:1\r\n:2\r\n:3\r\n => ["1", "2", "3"]
func readArray(data []byte) ([]interface{}, int, error) {
	length, lastPos, _ := readLen(data)

	var array = make([]interface{}, length)

	for i := range length {
		value, pos, _ := DecodeOne(data)
		array[i] = value
		lastPos = pos
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
