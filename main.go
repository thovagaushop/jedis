package main

import (
	"jedis/internal/server"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(2)
	var signals = make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	go server.RunAsyncTCPServer()
	go server.WaitForSignal(&wg, signals)

	wg.Wait()
}
