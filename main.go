package main

import "jedis/internal/server"

func main() {
	server.RunAsyncTCPServer()
}
