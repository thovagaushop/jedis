package server

import (
	"fmt"
	"jedis/config"
	"jedis/pkg/ringbuffer"
	"sync"
	"sync/atomic"
	"time"
)

func Initializing() error {
	var wg sync.WaitGroup
	cpus := config.MAX_CPU_CORE
	shardingThreads = make([]*ShardingThread, cpus)

	// Initializing partition
	for i := 0; i < cpus; i++ {
		shardingThreads[i] = NewShardingThread(i)
	}

	// Create thread pipeline for partition
	for i := 0; i < cpus; i++ {
		for j := 0; j < cpus; j++ {
			if i == j {
				continue
			}

			inputRb, err := ringbuffer.NewRingBuffer[Task](262144)
			if err != nil {
				return err
			}

			outputRb, err := ringbuffer.NewRingBuffer[Task](262144)

			if err != nil {
				return err
			}

			shardingThreads[i].OutboundQueue[j] = outputRb
			shardingThreads[j].InboundQueue[i] = inputRb
		}
	}

	// Run worker
	for i := range cpus {
		wg.Add(1)
		go func(idx int) {
			if err := shardingThreads[idx].Run(&wg); err != nil {
				fmt.Printf("Thread %d exited with error: %v\n", idx, err)
			}
		}(i)
	}

	go func() {
		for {
			time.Sleep(2 * time.Second)
			fmt.Println("--- Load Balance Stats (Every 2s) ---")
			for i := 0; i < cpus; i++ {
				c := atomic.SwapUint64(&shardingThreads[i].processedCmds, 0)
				a := atomic.SwapUint64(&shardingThreads[i].acceptedConns, 0)
				if c > 0 || a > 0 {
					fmt.Printf("Thread %d: accepted=%d cmds=%d\n", i, a, c)
				}
			}
		}
	}()

	wg.Wait()
	return nil
}
