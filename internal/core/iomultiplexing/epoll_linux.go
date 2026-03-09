//go:build linux

package iomultiplexing

import (
	"jedis/config"

	"golang.org/x/sys/unix"
)

type Epoll struct {
	Fd            int32
	epollEvents   []unix.EpollEvent
	genericEvents []Event
}

func CreateIOMultiplexing() (IOMultiplexing, error) {
	epfd, err := unix.EpollCreate1(0)

	if err != nil {
		return nil, err
	}

	return &Epoll{
		Fd:            int32(epfd),
		epollEvents:   make([]unix.EpollEvent, config.MAX_CONNECTION),
		genericEvents: make([]Event, config.MAX_CONNECTION),
	}, nil
}

func (ep *Epoll) Register(event Event) error {
	var epollEvent unix.EpollEvent
	epollEvent.Fd = event.Fd
	epollEvent.Events = unix.EPOLLIN
	if err := unix.EpollCtl(int(ep.Fd), unix.EPOLL_CTL_ADD, int(event.Fd), &epollEvent); err != nil {
		return err
	}
	return nil
}

func (ep *Epoll) Check() ([]Event, error) {
	nevents, err := unix.EpollWait(int(ep.Fd), ep.epollEvents, -1)

	if err != nil {
		return nil, err
	}

	for i := 0; i < nevents; i++ {
		ep.genericEvents[i] = createEvent(ep.epollEvents[i])
	}

	return ep.genericEvents[:nevents], nil
}

func (ep *Epoll) Close() error {
	return unix.Close(int(ep.Fd))
}
