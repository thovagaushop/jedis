//go:build darwin

package iomultiplexing

import "golang.org/x/sys/unix"

func (e Event) toNative(flags uint16) unix.Kevent_t {
	var event unix.Kevent_t

	if e.OpCode&OpcodeRead != 0 {
		event.Filter = unix.EVFILT_READ
	}
	if e.OpCode&OpcodeWrite != 0 {
		event.Filter = unix.EVFILT_WRITE
	}

	event.Ident = uint64(e.Fd)
	event.Flags = flags
	return event
}

func createEvent(kEvent unix.Kevent_t) Event {
	var event Event
	var op OpcodeType

	if kEvent.Filter == unix.EVFILT_READ {
		op = OpcodeRead
	}
	if kEvent.Filter == unix.EVFILT_WRITE {
		op = OpcodeWrite
	}
	event.Fd = int32(kEvent.Ident)
	event.OpCode = op
	return event
}
