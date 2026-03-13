//go:build darwin

package iomultiplexing

import (
	"jedis/config"

	"golang.org/x/sys/unix"
)

type Kqueue struct {
	Fd            int32
	kqueueEvents  []unix.Kevent_t
	genericEvents []Event
}

func CreateIOMultiplexing() (IOMultiplexing, error) {
	kqueueFd, err := unix.Kqueue()

	if err != nil {
		return nil, err
	}

	return &Kqueue{
		Fd:            int32(kqueueFd),
		kqueueEvents:  make([]unix.Kevent_t, config.MAX_CONNECTION),
		genericEvents: make([]Event, config.MAX_CONNECTION),
	}, nil
}

func (k *Kqueue) Register(event Event) error {
	kEvent := event.toNative(unix.EV_ADD)

	if _, err := unix.Kevent(int(k.Fd), []unix.Kevent_t{kEvent}, nil, nil); err != nil {
		return err
	}
	return nil
}

func (k *Kqueue) Modify(event Event) error {
	kevs := event.toNative(unix.EV_ADD)

	if _, err := unix.Kevent(int(k.Fd), []unix.Kevent_t{kevs}, nil, nil); err != nil {
		return err
	}
	return nil
}

func (k *Kqueue) Check() ([]Event, error) {
	n, err := unix.Kevent(int(k.Fd), nil, k.kqueueEvents, nil)

	if err != nil {
		return nil, err
	}
	for i := 0; i < n; i++ {
		k.genericEvents[i] = createEvent(k.kqueueEvents[i])
	}

	return k.genericEvents[:n], nil
}

func (k *Kqueue) Close() error {
	return unix.Close(int(k.Fd))
}
