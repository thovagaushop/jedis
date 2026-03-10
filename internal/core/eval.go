package core

import "jedis/internal/constant"

func EvalAndResponse(cmd *JedisCmd) []byte {
	var res []byte

	switch cmd.Cmd {
	case "PING":
		res = []byte("+PONG\r\n")
	case "SET":
		res = cmdSet(cmd.Args)

	case "GET":
		res = cmdGet(cmd.Args)
	default:
		res = []byte(constant.RESP_OK)
	}

	return res
}
