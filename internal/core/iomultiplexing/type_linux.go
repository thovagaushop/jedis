//go:build linux

package iomultiplexing

import "golang.org/x/sys/unix"

func (e Event) toNative() unix.EpollEvent {
	var event unix.EpollEvent

	if e.OpCode&OpcodeRead != 0 {
		event.Events |= unix.EPOLLIN
	}
	if e.OpCode&OpcodeWrite != 0 {
		event.Events |= unix.EPOLLOUT
	}

	event.Fd = e.Fd
	return event
}

func createEvent(epollEvent unix.EpollEvent) Event {
	var event Event
	var op OpcodeType

	if epollEvent.Events&unix.EPOLLIN != 0 {
		op |= OpcodeRead
	}
	if epollEvent.Events&unix.EPOLLOUT != 0 {
		op |= OpcodeWrite
	}
	event.Fd = epollEvent.Fd
	event.OpCode = op
	return event
}
