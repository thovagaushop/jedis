//go:build linux

package server

import (
	"fmt"
	"jedis/config"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

const (
	EPOLLET        = 1 << 31
	MAX_CONNECTION = 32
)

func echo(fd int) {
	// defer unix.Close(fd)
	var buf [32 * 1024]byte
	for {
		nbytes, e := unix.Read(fd, buf[:])
		if nbytes > 0 {
			fmt.Printf(">>> %s", buf)
			unix.Write(fd, buf[:nbytes])
			fmt.Printf("<<< %s", buf)
			continue
		}

		if nbytes == 0 {
			// Client disconnected
			fmt.Printf("client disconnected")
			unix.Close(fd)
			return
		}
		if e != nil {
			if e == unix.EAGAIN || e == unix.EWOULDBLOCK {
				return
			}
			fmt.Printf("error when echo: %v", e)
			unix.Close(fd)
			return
		}
	}
}

func RunTCPAsynchrousServer() {
	var event unix.EpollEvent
	var events [MAX_CONNECTION]unix.EpollEvent

	fd, err := unix.Socket(unix.AF_INET, unix.O_NONBLOCK|unix.SOCK_STREAM, 0)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer unix.Close(fd)

	// Set non block here
	if err := unix.SetNonblock(fd, true); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Add this fd to an address with specific port and Ip
	addr := unix.SockaddrInet4{Port: config.GlobalConfig.Port}
	copy(addr.Addr[:], net.ParseIP(config.GlobalConfig.Host).To4())

	// Bind this fd to this address
	unix.Bind(fd, &addr)
	unix.Listen(fd, MAX_CONNECTION)

	// Create an epoll to register the socket fd to it
	epfd, err := unix.EpollCreate1(0)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Register the main socket fd to epoll
	event.Fd = int32(fd)
	event.Events = unix.EPOLLIN

	if err := unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fd, &event); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Running jedis server")
	// The main loop
	for {
		// We will be waiting for the new event from epoll
		nevents, err := unix.EpollWait(epfd, events[:], -1)
		if err != nil {
			fmt.Println(err)
			break
		}

		for i := 0; i < nevents; i++ {
			ev := events[i]

			// There are 2 condition
			// 1. If the main socket fd have new data, we'll have new connection
			// 2. Else, the connection fd have new data

			if ev.Fd == int32(fd) {
				// We will create new connection
				connFd, _, err := unix.Accept(fd)

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
				event.Fd = int32(connFd)
				event.Events = unix.EPOLLIN | unix.EPOLLET

				if err := unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, connFd, &event); err != nil {
					fmt.Printf("error when register connection fd to epoll: %v", err)
					os.Exit(1)
				}
			} else {
				go echo(int(ev.Fd))
			}
		}
	}

}
