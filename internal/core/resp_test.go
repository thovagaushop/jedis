package core

import (
	"fmt"
	"testing"
)

type Testcase struct {
	Input    string
	Type     string
	Expected any
}

func TestRespReadLine(t *testing.T) {
	s := "hello"
	s1 := []string{
		"string",
		"hello",
	}

	res := encodeSimpleString(s)
	fmt.Println(res)
	res1 := encodeStringArray(s1)
	fmt.Println(res1)
}
