package server

import (
	"fmt"
	"hash/crc32"
	"jedis/config"
	"jedis/internal/core"
	"jedis/internal/core/iomultiplexing"
	"jedis/pkg/ringbuffer"
	"runtime"
	"sync"
	"sync/atomic"

	"golang.org/x/sys/unix"
)

var shardingThreads []*ShardingThread

type Task struct {
	ConnFd int
	Data   []byte
}

type ShardingThread struct {
	Id            int
	InboundQueue  []*ringbuffer.RingBuffer[Task]
	OutboundQueue []*ringbuffer.RingBuffer[Task]
	iom           iomultiplexing.IOMultiplexing
	processedCmds uint64
	acceptedConns uint64
}

func getShardID(key *string, totalPartition int) int {
	if key == nil {
		return -1
	}
	checkSum := crc32.ChecksumIEEE([]byte(*key))
	partition := int(checkSum % uint32(totalPartition))
	return partition
}

func NewShardingThread(id int) *ShardingThread {
	ioMuliplexing, err := iomultiplexing.CreateIOMultiplexing()
	if err != nil {
		fmt.Printf("create io multiplexing error: %v\n", err)
		return nil
	}

	return &ShardingThread{
		Id:            id,
		InboundQueue:  make([]*ringbuffer.RingBuffer[Task], config.MAX_CPU_CORE),
		OutboundQueue: make([]*ringbuffer.RingBuffer[Task], config.MAX_CPU_CORE),
		iom:           ioMuliplexing,
	}
}

func (p *ShardingThread) handleIO(socketFd int) (bool, error) {
	events, err := p.iom.Check(10)
	if err != nil {
		return false, err
	}

	for _, ev := range events {
		if ev.Fd == int32(socketFd) {
			// If this fd is equal to socket fd, that means there are new connection
			// 1. Accept this connection and return connection fd
			// 2. Set this connection fd to non blocking
			// 3. Register it to epoll for receiving new data sent from client
			connFd, _, err := unix.Accept(socketFd)
			if err != nil {
				fmt.Printf("Accept error: %v\n", err)
				return true, err
			}

			if err := unix.SetNonblock(connFd, true); err != nil {
				unix.Close(connFd)
				return true, err
			}

			if err := p.iom.Register(iomultiplexing.Event{
				Fd:     int32(connFd),
				OpCode: iomultiplexing.OpcodeRead,
			}); err != nil {
				unix.Close(connFd)
				return true, err
			}
			atomic.AddUint64(&p.acceptedConns, 1)
		} else {
			var buf = make([]byte, 512)
			n, err := unix.Read(int(ev.Fd), buf)
			if err != nil {
				return false, err
			}

			cmd, _, err := core.ParseCmd(buf[:n])
			cmm := core.FDComm{Fd: int(ev.Fd)}

			if err != nil {
				unix.Close(int(ev.Fd))
				// fmt.Printf("client closed: %d\r\n", ev.Fd)
				continue
			}
			atomic.AddUint64(&p.processedCmds, 1)

			// Process CMD
			key := cmd.Key
			partitionId := getShardID(key, config.MAX_CPU_CORE)

			if partitionId == p.Id || partitionId == -1 {
				responseRw(cmd, cmm)
			} else {
				// fmt.Printf("Thread %d forward a command to thread %d\n", p.Id, partitionId)
				partition := shardingThreads[partitionId]
				for {
					if ok := partition.InboundQueue[p.Id].Push(Task{
						ConnFd: int(ev.Fd),
						Data:   buf[:n],
					}); !ok {
						runtime.Gosched()
						continue
					}
					break
				}
			}
		}
	}
	return true, nil
}

func (p *ShardingThread) handleInternalQueue() error {
	for range config.MAX_LOOP_TIME {
		for i := range config.MAX_CPU_CORE {
			if i == p.Id {
				continue
			}
			task, ok := p.InboundQueue[i].Pop()
			if !ok {
				continue
			}
			// fmt.Printf("Thread %d receive a command from thread %d\n", p.Id, i)
			cmd, _, err := core.ParseCmd(task.Data)
			if err != nil {
				continue
			}
			res := core.EvalAndResponse(cmd)
			task.Data = res
			targetThread := shardingThreads[i]
			for {
				if !targetThread.OutboundQueue[p.Id].Push(task) {
					runtime.Gosched()
					continue
				}
				break
			}
		}
	}

	// Handle outbound
	for range config.MAX_LOOP_TIME {
		for i := range config.MAX_CPU_CORE {
			if i == p.Id {
				continue
			}
			task, ok := p.OutboundQueue[i].Pop()

			if !ok {
				continue
			}

			// fmt.Printf("Thread %d receive a response from thread %d\n", p.Id, i)
			cmm := core.FDComm{Fd: task.ConnFd}
			_, err := cmm.Write(task.Data)

			if err != nil {
				cmm.Write([]byte(fmt.Sprintf("-%s%s", err, core.CRLF)))
			}
		}
	}
	return nil
}

func (p *ShardingThread) Run(wg *sync.WaitGroup) error {
	runtime.LockOSThread()
	defer func() {
		runtime.UnlockOSThread()
		p.iom.Close()
		wg.Done()
	}()

	fd, err := bindSocket()
	if err != nil {
		fmt.Println("bind socket error: ", err)
		return err
	}
	defer unix.Close(fd)

	// Register the socket fd
	if err := p.iom.Register(iomultiplexing.Event{
		Fd:     int32(fd),
		OpCode: iomultiplexing.OpcodeRead,
	}); err != nil {
		fmt.Printf("register fd error: %v\n", err)
		return err
	}

	fmt.Printf("Running jedis thread on partition %d\n", p.Id)

	// Decide to spin loop here for reducing context switch
	// I think this is a trade-off between latency and cpu usage
	for {
		// 1. Handle I/O
		ok, err := p.handleIO(fd)

		if !ok {
			return err
		}

		// 2. Handle internal queue
		if err := p.handleInternalQueue(); err != nil {
			return err
		}
	}
}
