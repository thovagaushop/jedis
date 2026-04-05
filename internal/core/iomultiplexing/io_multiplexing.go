package iomultiplexing

type OpcodeType int

const (
	OpcodeRead  OpcodeType = 1 << 0 // 1
	OpcodeWrite OpcodeType = 1 << 1 // 2
)

type Event struct {
	Fd     int32
	OpCode OpcodeType
}

type IOMultiplexing interface {
	Register(event Event) error
	Modify(event Event) error
	Check(msTimeout int64) ([]Event, error)
	Close() error
}
