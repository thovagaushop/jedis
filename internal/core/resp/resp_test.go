package resp

import (
	"testing"
)

type Testcase struct {
	Input    string
	Type     string
	Expected any
}

func TestRespReadLine(t *testing.T) {

	// testcases := []Testcase{
	// 	{
	// 		Input:    "+PING\r\n",
	// 		Type:     "STRING",
	// 		Expected: "PING",
	// 	},
	// 	{
	// 		Input:    ":123\r\n",
	// 		Type:     "INTEGER",
	// 		Expected: int64(123),
	// 	},
	// 	{
	// 		Input:    "$5\r\nhello\r\n",
	// 		Type:     "BULK",
	// 		Expected: "hello",
	// 	},
	// }

	// for _, tc := range testcases {
	// 	switch tc.Type {
	// 	case "STRING":
	// 		string, _ := readSimpleString([]byte(tc.Input))
	// 		if string != tc.Expected {
	// 			t.Error("wrong")
	// 		}
	// 	case "INTEGER":
	// 		value, _ := readInt64([]byte(tc.Input))
	// 		if value != tc.Expected {
	// 			t.Error("wrong")
	// 		}
	// 	case "BULK":
	// 		value, pos := readBulkString([]byte(tc.Input))
	// 		log.Println(value, " ", pos)
	// 	}
	// }
}
