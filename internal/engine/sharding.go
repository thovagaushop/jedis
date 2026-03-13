package engine

import (
	"hash/crc32"
	"jedis/config"
	"jedis/internal/core"
	"jedis/internal/server"
	"os"
	"sync"
)

func getShardID(key string, totalPartition int) int {
	checkSum := crc32.ChecksumIEEE([]byte(key))
	partition := int(checkSum % uint32(totalPartition))
	return partition
}

type Worker struct {
	id       int
	shareMem chan core.JedisCmd
}

func NewWorker(id int) *Worker {
	return &Worker{
		id:       id,
		shareMem: make(chan core.JedisCmd, config.MAX_CONNECTION),
	}
}

func (w *Worker) Run() error {
	err := server.RunAsyncTCPServer()
	if err != nil {
		return err
	}

	return nil
}

type ShardingEngine struct {
	cpus int
}

func NewShardingEngine(cpus int) IEngine {
	return &ShardingEngine{}
}

func (e *ShardingEngine) Run(wg *sync.WaitGroup, signals chan os.Signal) {
	wg.Add(e.cpus + 1)
	for i := range e.cpus {
		worker := NewWorker(i)
		go worker.Run()
	}
}
