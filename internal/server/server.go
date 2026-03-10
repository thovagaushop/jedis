//go:build linux

package server

import (
	"fmt"
	"jedis/config"
	"jedis/internal/core"
	"jedis/internal/core/iomultiplexing"
	"net"

	"golang.org/x/sys/unix"
)

func readCommand(fd int) (*core.JedisCmd, error) {
	buf := make([]byte, 512)

	n, err := unix.Read(fd, buf)
	if err != nil {
		return nil, err
	}
	return core.ParseCmd(buf[:n])
}

func responseCommand(fd int, cmd *core.JedisCmd) error {
	_, e := unix.Write(fd, core.EvalAndResponse(cmd))
	if e != nil {
		return e
	}
	return nil
}

func RunAsyncTCPServer() error {
	// var event unix.EpollEvent
	// var events [config.MAX_CONNECTION]iomultiplexing.Event
	fd, err := unix.Socket(unix.AF_INET, unix.O_NONBLOCK|unix.SOCK_STREAM, 0)

	if err != nil {
		return err
	}
	defer unix.Close(fd)

	// Set non block here
	if err := unix.SetNonblock(fd, true); err != nil {
		return err
	}

	// Add this fd to an address with specific port and Ip
	addr := unix.SockaddrInet4{Port: config.PORT}
	copy(addr.Addr[:], net.ParseIP(config.HOST).To4())

	// Bind this fd to this address
	unix.Bind(fd, &addr)
	unix.Listen(fd, config.MAX_CONNECTION)

	// Create an epoll to register the socket fd to it
	// epfd, err := unix.EpollCreate1(0)

	// if err != nil {
	// 	return err
	// }

	// Create IOMultiplexing
	ioMultiplexing, err := iomultiplexing.CreateIOMultiplexing()

	// Register the main socket fd to epoll
	if err := ioMultiplexing.Register(iomultiplexing.Event{
		Fd: int32(fd),
	}); err != nil {
		return err
	}

	fmt.Println("Running jedis server")
	// The main loop
	for {
		// We will be waiting for the new event from epoll
		nevents, err := ioMultiplexing.Check()
		// nevents, err := unix.EpollWait(epfd, events[:], -1)
		if err != nil {
			return err
		}

		for i := 0; i < len(nevents); i++ {
			// ev := events[i]

			ev := nevents[i]

			// There are 2 condition
			// 1. If the main socket fd have new data, we'll have new connection
			// 2. Else, the connection fd have new data

			if ev.Fd == int32(fd) {
				// We will create new connection
				connFd, _, err := unix.Accept(fd)
				fmt.Printf("new connection: %d\n", connFd)

				if err != nil {
					fmt.Printf("error when create new connection: %v", err)
					continue
				}

				// Set non block for this fd as well
				if err := unix.SetNonblock(connFd, true); err != nil {
					fmt.Printf("error when set non block for connection: %v", err)
					continue
				}

				// We register connFd to epoll as well
				if err := ioMultiplexing.Register(iomultiplexing.Event{
					Fd: int32(connFd),
				}); err != nil {
					return err
				}
			} else {
				go readCommand(int(ev.Fd))
				cmd, err := readCommand(int(ev.Fd))

				if err != nil {
					unix.Close(int(ev.Fd))
					continue
				}

				go responseCommand(int(ev.Fd), cmd)

			}
		}
	}

}
