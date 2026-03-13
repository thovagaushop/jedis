package server

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

func TestAsyncServer_Integration(t *testing.T) {
	// Start server in background
	go func() {
		if err := RunAsyncTCPServer(); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	t.Run("BasicCommands", func(t *testing.T) {
		conn, err := net.Dial("tcp", "127.0.0.1:6379")
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()

		// Test PING
		conn.Write([]byte("*1\r\n$4\r\nPING\r\n"))
		reader := bufio.NewReader(conn)
		line, _ := reader.ReadString('\n')
		if line != "+PONG\r\n" {
			t.Errorf("expected +PONG\\r\\n, got %q", line)
		}

		// Test SET
		conn.Write([]byte("*3\r\n$3\r\nSET\r\n$4\r\nname\r\n$5\r\njedis\r\n"))
		line, _ = reader.ReadString('\n')
		if line != "+OK\r\n" {
			t.Errorf("expected +OK\\r\\n, got %q", line)
		}

		// Test GET
		conn.Write([]byte("*2\r\n$3\r\nGET\r\n$4\r\nname\r\n"))
		line, _ = reader.ReadString('\n')
		if line != "$5\r\n" {
			t.Errorf("expected $5\\r\\n (bulk string len), got %q", line)
		}
		line, _ = reader.ReadString('\n')
		if line != "jedis\r\n" {
			t.Errorf("expected jedis\\r\\n, got %q", line)
		}
	})

	t.Run("ConcurrentClients", func(t *testing.T) {
		const numClients = 1000
		var wg sync.WaitGroup
		wg.Add(numClients)

		for i := 0; i < numClients; i++ {
			go func(id int) {
				defer wg.Done()
				conn, err := net.Dial("tcp", "127.0.0.1:6379")
				if err != nil {
					t.Errorf("Client %d failed to connect: %v", id, err)
					return
				}
				defer conn.Close()

				key := fmt.Sprintf("key-%d", id)
				val := fmt.Sprintf("val-%d", id)

				// SET
				conn.Write([]byte(fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(key), key, len(val), val)))
				reader := bufio.NewReader(conn)
				reader.ReadString('\n') // read +OK

				// GET
				conn.Write([]byte(fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(key), key)))
				reader.ReadString('\n') // read bulk len
				result, _ := reader.ReadString('\n')
				if result != val+"\r\n" {
					t.Errorf("Client %d expected %s, got %s", id, val, result)
				}
			}(i)
		}
		wg.Wait()
	})

	t.Run("Pipelining", func(t *testing.T) {
		conn, err := net.Dial("tcp", "127.0.0.1:6379")
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()

		// Send 3 PINGs at once
		conn.Write([]byte("*1\r\n$4\r\nPING\r\n*1\r\n$4\r\nPING\r\n*1\r\n$4\r\nPING\r\n"))

		reader := bufio.NewReader(conn)
		for i := 0; i < 3; i++ {
			line, _ := reader.ReadString('\n')
			if line != "+PONG\r\n" {
				t.Errorf("pipelined PING %d: expected +PONG, got %q", i, line)
			}
		}
	})

	t.Run("StressRaceCondition", func(t *testing.T) {
		const numClients = 100
		const opsPerClient = 200
		var wg sync.WaitGroup
		wg.Add(numClients)

		for i := 0; i < numClients; i++ {
			go func(cid int) {
				defer wg.Done()
				conn, err := net.Dial("tcp", "127.0.0.1:6379")
				if err != nil {
					t.Errorf("Client %d failed to connect: %v", cid, err)
					return
				}
				defer conn.Close()

				reader := bufio.NewReader(conn)
				for j := 0; j < opsPerClient; j++ {
					// Mix of shared and unique keys
					var key string
					if j%2 == 0 {
						key = "shared-key"
					} else {
						key = fmt.Sprintf("unique-key-%d-%d", cid, j)
					}

					val := fmt.Sprintf("v-%d-%d", cid, j)

					// SET
					setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(key), key, len(val), val)
					_, err := conn.Write([]byte(setCmd))
					if err != nil {
						return
					}
					_, _ = reader.ReadString('\n') // +OK

					// GET
					getCmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(key), key)
					_, err = conn.Write([]byte(getCmd))
					if err != nil {
						return
					}
					line1, _ := reader.ReadString('\n') // bulk len
					if line1 == "$-1\r\n" {
						// Key might have been overwritten or something if shared, but should exist
						continue
					}
					_, _ = reader.ReadString('\n') // value
				}
			}(i)
		}
		wg.Wait()
	})
}
