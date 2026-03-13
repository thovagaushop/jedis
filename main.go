package main

import (
	"jedis/internal/engine"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	var wg sync.WaitGroup
	var signals = make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	engine := engine.NewSingleThreadEngine()
	engine.Run(&wg, signals)
	wg.Wait()
}
