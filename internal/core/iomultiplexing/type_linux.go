package iomultiplexing

import "golang.org/x/sys/unix"

func (e Event) toNative() unix.EpollEvent {
	var event unix.EpollEvent

	event.Fd = e.Fd
	return event
}

func createEvent(epollEvent unix.EpollEvent) Event {
	var event Event

	event.Fd = epollEvent.Fd
	return event
}
