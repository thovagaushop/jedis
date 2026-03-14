package server

import (
	"fmt"
	"hash/crc32"
	"io"
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

func readCommandFD(fd int) (*core.JedisCmd, error) {
	var buf = make([]byte, 512)
	n, err := unix.Read(fd, buf)
	if err != nil {
		return nil, err
	}
	cmd, _, err := core.ParseCmd(buf[:n])
	return cmd, err
}

func responseRw(cmd *core.JedisCmd, rw io.ReadWriter) {
	err := core.EvalAndResponse(cmd, rw)
	if err != nil {
		responseErrorRw(err, rw)
	}
}

func responseErrorRw(err error, rw io.ReadWriter) {
	rw.Write([]byte(fmt.Sprintf("-%s%s", err, core.CRLF)))
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
				fmt.Printf("New connection: %d\n", connFd)
			} else {
				cmm := core.FDComm{Fd: int(ev.Fd)}
				cmd, err := readCommandFD(int(ev.Fd))

				if err != nil {
					unix.Close(int(ev.Fd))
					fmt.Printf("client closed: %d\r\n", ev.Fd)
					atomic.SwapInt32(&eStatus, constant.EngineStatusWaiting)
					continue
				}

				responseRw(cmd, cmm)
			}
			atomic.SwapInt32(&eStatus, constant.EngineStatusWaiting)
		}
	}

	return nil
}

func getShardID(key string, totalPartition int) int {
	checkSum := crc32.ChecksumIEEE([]byte(key))
	partition := int(checkSum % uint32(totalPartition))
	return partition
}
