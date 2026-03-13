package server

import (
	"fmt"
	"jedis/config"
	"jedis/internal/constant"
	"jedis/internal/core"
	"jedis/internal/core/iomultiplexing"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"

	"golang.org/x/sys/unix"
)

var eStatus int32 = constant.EngineStatusWaiting

func WaitForSignal(wg *sync.WaitGroup, signals chan os.Signal) {
	defer wg.Done()

	c := <-signals
	log.Println(c)
	for atomic.LoadInt32(&eStatus) == constant.EngineStatusBusy {
	}
	log.Println("Shutting down gracefully")
	os.Exit(0)
}

type Client struct {
	fd         int
	inputBuff  []byte
	outputBuff []byte
}

var clientMapping map[int]*Client

func newClient(fd int) *Client {
	client := &Client{
		fd:         fd,
		inputBuff:  make([]byte, 0),
		outputBuff: make([]byte, 0),
	}
	clientMapping[fd] = client
	return client
}

func delClient(fd int) {
	delete(clientMapping, fd)
	unix.Close(fd)
}

func handleRead(ioMultiplexing iomultiplexing.IOMultiplexing, fd int) error {
	client := clientMapping[fd]
	buf := make([]byte, 1024)
	n, err := unix.Read(fd, buf)
	if err != nil {
		if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
			return nil
		}
		return err
	}
	if n == 0 {
		return fmt.Errorf("client closed")
	}

	client.inputBuff = append(client.inputBuff, buf[:n]...)

	// Loop to parse all commands available in the buffer (Pipelining)
	for len(client.inputBuff) > 0 {
		cmd, pos, err := core.ParseCmd(client.inputBuff)
		if err != nil {
			// Incomplete command, wait for more data
			break
		}

		// Consume parsed bytes from buffer
		client.inputBuff = client.inputBuff[pos:]

		// Execute
		res := core.EvalAndResponse(cmd)
		client.outputBuff = append(client.outputBuff, res...)
	}

	return flushOutput(ioMultiplexing, client)
}

func flushOutput(ioMultiplexing iomultiplexing.IOMultiplexing, client *Client) error {
	if len(client.outputBuff) == 0 {
		return ioMultiplexing.Modify(iomultiplexing.Event{
			Fd:     int32(client.fd),
			OpCode: iomultiplexing.OpcodeRead,
		})
	}

	n, err := unix.Write(client.fd, client.outputBuff)
	if err != nil {
		if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
			// Kernel buffer full, wait for EPOLLOUT
			return ioMultiplexing.Modify(iomultiplexing.Event{
				Fd:     int32(client.fd),
				OpCode: iomultiplexing.OpcodeRead | iomultiplexing.OpcodeWrite,
			})
		}
		return err
	}

	client.outputBuff = client.outputBuff[n:]

	if len(client.outputBuff) == 0 {
		// Done writing, stop listening for EPOLLOUT
		return ioMultiplexing.Modify(iomultiplexing.Event{
			Fd:     int32(client.fd),
			OpCode: iomultiplexing.OpcodeRead,
		})
	} else {
		// Still have data, ensure we listen for EPOLLOUT
		return ioMultiplexing.Modify(iomultiplexing.Event{
			Fd:     int32(client.fd),
			OpCode: iomultiplexing.OpcodeRead | iomultiplexing.OpcodeWrite,
		})
	}
}

func RunAsyncTCPServer() error {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	if err := unix.SetNonblock(fd, true); err != nil {
		return err
	}

	addr := unix.SockaddrInet4{Port: config.PORT}
	copy(addr.Addr[:], net.ParseIP(config.HOST).To4())

	if err := unix.Bind(fd, &addr); err != nil {
		return err
	}
	if err := unix.Listen(fd, config.MAX_CONNECTION); err != nil {
		return err
	}

	ioMultiplexing, err := iomultiplexing.CreateIOMultiplexing()
	if err != nil {
		return err
	}

	if err := ioMultiplexing.Register(iomultiplexing.Event{
		Fd:     int32(fd),
		OpCode: iomultiplexing.OpcodeRead,
	}); err != nil {
		return err
	}

	clientMapping = make(map[int]*Client)

	fmt.Println("Running jedis server (Single-threaded Event Loop)")
	for atomic.LoadInt32(&eStatus) != constant.EngineStatusShuttingDown {
		events, err := ioMultiplexing.Check()
		if err != nil {
			return err
		}

		if !atomic.CompareAndSwapInt32(&eStatus, constant.EngineStatusWaiting, constant.EngineStatusBusy) {
			if eStatus == constant.EngineStatusShuttingDown {
				return nil
			}
		}

		for _, ev := range events {
			if ev.Fd == int32(fd) {
				connFd, _, err := unix.Accept(fd)
				if err != nil {
					fmt.Printf("Accept error: %v\n", err)
					continue
				}

				if err := unix.SetNonblock(connFd, true); err != nil {
					unix.Close(connFd)
					continue
				}

				if err := ioMultiplexing.Register(iomultiplexing.Event{
					Fd:     int32(connFd),
					OpCode: iomultiplexing.OpcodeRead,
				}); err != nil {
					unix.Close(connFd)
					continue
				}
				newClient(connFd)
				fmt.Printf("New connection: %d\n", connFd)
			} else {
				client, ok := clientMapping[int(ev.Fd)]
				if !ok {
					continue
				}

				if ev.OpCode&iomultiplexing.OpcodeRead != 0 {
					if err := handleRead(ioMultiplexing, int(ev.Fd)); err != nil {
						fmt.Printf("Closing client %d: %v\n", ev.Fd, err)
						delClient(int(ev.Fd))
						atomic.SwapInt32(&eStatus, constant.EngineStatusWaiting)
						continue
					}
				}

				// Only if client still exists
				if _, ok := clientMapping[int(ev.Fd)]; ok {
					if ev.OpCode&iomultiplexing.OpcodeWrite != 0 {
						if err := flushOutput(ioMultiplexing, client); err != nil {
							fmt.Printf("Write error on client %d: %v\n", ev.Fd, err)
							delClient(int(ev.Fd))
							atomic.SwapInt32(&eStatus, constant.EngineStatusWaiting)
						}
					}
				}
			}
			atomic.SwapInt32(&eStatus, constant.EngineStatusWaiting)
		}
	}

	return nil
}

// func RunWorkerServer() error {

// }
