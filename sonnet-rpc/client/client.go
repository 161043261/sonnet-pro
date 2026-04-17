package client

import (
	"context"
	"io"
	"reflect"
	. "lark_rpc"
	"sync"
)

type ClientDemo struct {
	d       Discovery
	mode    SelectMode
	opt     *Option
	mu      sync.Mutex // protect following
	clients map[string]*Client
}

var _ io.Closer = (*ClientDemo)(nil)

func NewClientDemo(d Discovery, mode SelectMode, opt *Option) *ClientDemo {
	return &ClientDemo{d: d, mode: mode, opt: opt, clients: make(map[string]*Client)}
}

func (self *ClientDemo) Close() error {
	self.mu.Lock()
	defer self.mu.Unlock()
	for key, client := range self.clients {
		// I have no idea how to deal with error, just ignore it.
		_ = client.Close()
		delete(self.clients, key)
	}
	return nil
}

func (self *ClientDemo) dial(rpcAddr string) (*Client, error) {
	self.mu.Lock()
	defer self.mu.Unlock()
	client, ok := self.clients[rpcAddr]
	if ok && !client.IsAvailable() {
		_ = client.Close()
		delete(self.clients, rpcAddr)
		client = nil
	}
	if client == nil {
		var err error
		client, err = XDial(rpcAddr, self.opt)
		if err != nil {
			return nil, err
		}
		self.clients[rpcAddr] = client
	}
	return client, nil
}

func (self *ClientDemo) call(rpcAddr string, ctx context.Context, serviceMethod string, args, reply any) error {
	client, err := self.dial(rpcAddr)
	if err != nil {
		return err
	}
	return client.Call(ctx, serviceMethod, args, reply)
}

// Call invokes the named function, waits for it to complete,
// and returns its error status.
// self will choose a proper server.
func (self *ClientDemo) Call(ctx context.Context, serviceMethod string, args, reply any) error {
	rpcAddr, err := self.d.Get(self.mode)
	if err != nil {
		return err
	}
	return self.call(rpcAddr, ctx, serviceMethod, args, reply)
}

// Broadcast invokes the named function for every server registered in discovery
func (self *ClientDemo) Broadcast(ctx context.Context, serviceMethod string, args, reply any) error {
	servers, err := self.d.GetAll()
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	var mu sync.Mutex // protect e and replyDone
	var e error
	replyDone := reply == nil // if reply is nil, don't need to set value
	ctx, cancel := context.WithCancel(ctx)
	for _, rpcAddr := range servers {
		wg.Add(1)
		go func(rpcAddr string) {
			defer wg.Done()
			var clonedReply any
			if reply != nil {
				clonedReply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
			}
			err := self.call(rpcAddr, ctx, serviceMethod, args, clonedReply)
			mu.Lock()
			if err != nil && e == nil {
				e = err
				cancel() // if any call failed, cancel unfinished calls
			}
			if err == nil && !replyDone {
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(clonedReply).Elem())
				replyDone = true
			}
			mu.Unlock()
		}(rpcAddr)
	}
	wg.Wait()
	return e
}
