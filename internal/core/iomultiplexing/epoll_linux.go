//go:build linux

package iomultiplexing

import (
	"jedis/config"

	"golang.org/x/sys/unix"
)

type Epoll struct {
	Fd            int32
	eventFd       int32
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
	epollEvent := event.toNative()
	if err := unix.EpollCtl(int(ep.Fd), unix.EPOLL_CTL_ADD, int(event.Fd), &epollEvent); err != nil {
		return err
	}
	return nil
}

func (ep *Epoll) Modify(event Event) error {
	epollEvent := event.toNative()
	if err := unix.EpollCtl(int(ep.Fd), unix.EPOLL_CTL_MOD, int(event.Fd), &epollEvent); err != nil {
		return err
	}
	return nil
}

func (ep *Epoll) Check(msTimeout int64) ([]Event, error) {
	nevents, err := unix.EpollWait(int(ep.Fd), ep.epollEvents, int(msTimeout))

	if err != nil {
		if err == unix.EINTR {
			return nil, nil
		}
		return nil, err
	}

	events := make([]Event, nevents)

	for i := 0; i < nevents; i++ {
		events[i] = createEvent(ep.epollEvents[i])
	}

	return events, nil
}

func (ep *Epoll) Close() error {
	return unix.Close(int(ep.Fd))
}
