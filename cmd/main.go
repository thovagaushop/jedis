package main

import (
	"flag"
	"fmt"
	"jedis/config"
	"jedis/internal/server"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	mode := flag.String("mode", "sharding", "Execution Mode: 'single' or 'sharding'")
	cpus := flag.Int("cpus", 4, "Number of CPU Cores to use (Required >= 2 for sharding)")
	port := flag.Int("port", 6379, "Jedis Listening Port")
	host := flag.String("host", "0.0.0.0", "Jedis Host Address")
	maxConn := flag.Int("max-conn", 10000, "Maximum Connections Limit")

	flag.Parse()

	config.PORT = *port
	config.HOST = *host
	config.MAX_CONNECTION = *maxConn

	switch *mode {
	case "sharding":
		if *cpus < 2 {
			fmt.Println("❌ Error: 'sharding' mode (Multi-Thread) requires -cpus flag to be >= 2.")
			fmt.Println("👉 Example: ./jedis -mode=sharding -cpus=4")
			os.Exit(1)
		}
		config.MAX_CPU_CORE = *cpus
		fmt.Printf("🚀 Starting Jedis in SHARDING Mode (%d Cores)...\n", *cpus)
		server.Initializing()
	case "single":
		config.MAX_CPU_CORE = 1
		fmt.Printf("🐢 Starting Jedis in SINGLE-THREAD Mode (Host: %s:%d)...\n", *host, *port)
		
		var wg sync.WaitGroup
		var signals = make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

		wg.Add(1)
		go server.WaitForSignal(&wg, signals)

		if err := server.RunAsyncTCPServer(); err != nil {
			fmt.Printf("❌ Fatal Server Error: %v\n", err)
			os.Exit(1)
		}
		wg.Wait()
	default:
		fmt.Printf("❌ Error: Invalid mode '%s'. Please select either 'single' or 'sharding'.\n", *mode)
		os.Exit(1)
	}
}
