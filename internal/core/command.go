package core

import "golang.org/x/sys/unix"

type JedisCmd struct {
	Cmd  string
	Key  *string
	Args []string
}

type FDComm struct {
	Fd int
}

func (f FDComm) Read(data []byte) (int, error) {
	return unix.Read(f.Fd, data)
}

func (f FDComm) Write(data []byte) (int, error) {
	return unix.Write(f.Fd, data)
}
