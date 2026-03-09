package iomultiplexing

type Event struct {
	Fd int32
}

type IOMultiplexing interface {
	Register(ev Event) error
	Check() ([]Event, error)
	Close() error
}
