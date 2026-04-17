package transport

import (
	"errors"
	"net"
	"lark_rpc_v2/internal/protocol"
	"sync"
	"sync/atomic"
	"time"
)

type TCPClient struct {
	conn *TCPConnection
	addr string

	writeMu sync.Mutex
	seq     uint64

	pending sync.Map // map[uint64]*Future

	closed int32
}

func newTCPClient(addr string) (*TCPClient, error) {
	rawConn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, err
	}

	c := &TCPClient{
		conn: NewTCPConnection(rawConn),
		addr: addr,
	}

	go c.readLoop()
	return c, nil
}

func (c *TCPClient) nextSeq() uint64 {
	return atomic.AddUint64(&c.seq, 1)
}

func (c *TCPClient) SendAsync(msg *protocol.Message) (*Future, error) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return nil, errors.New("connection closed")
	}

	seq := c.nextSeq()
	msg.Header.RequestID = seq

	future := NewFuture()
	c.pending.Store(seq, future)

	c.writeMu.Lock()
	err := c.conn.Write(msg)
	c.writeMu.Unlock()

	if err != nil {
		c.pending.Delete(seq)
		c.fail(err) // Critical: kill connection completely even if write fails (fix previous connection bug)
		return nil, err
	}

	return future, nil
}

func (c *TCPClient) readLoop() {
	for {
		msg, err := c.conn.Read()
		if err != nil {
			c.fail(err)
			return
		}

		seq := msg.Header.RequestID

		val, ok := c.pending.LoadAndDelete(seq)
		if !ok {
			continue
		}

		future := val.(*Future)

		if msg.Header.Error != "" {
			future.Done(nil, errors.New(msg.Header.Error))
		} else {
			future.Done(msg.Body, nil)
		}
	}
}

func (c *TCPClient) fail(err error) {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return
	}

	// Close underlying connection
	// log.Println("Underlying connection closed")
	_ = c.conn.Close()

	// Fail all pending requests
	c.pending.Range(func(key, value any) bool {
		future := value.(*Future)
		future.Done(nil, err)
		c.pending.Delete(key)
		return true
	})
}

func (c *TCPClient) Close() error {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return nil
	}
	return c.conn.Close()
}
