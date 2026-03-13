package engine

import (
	"jedis/internal/server"
	"os"
	"sync"
)

type SingleThreadEngine struct {
}

func NewSingleThreadEngine() IEngine {
	return &SingleThreadEngine{}
}

func (e *SingleThreadEngine) Run(wg *sync.WaitGroup, signals chan os.Signal) {
	wg.Add(2)
	go server.RunAsyncTCPServer()
	go server.WaitForSignal(wg, signals)
}
