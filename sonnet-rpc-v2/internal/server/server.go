package server

import (
	"log"
	"net"
	"lark_rpc_v2/internal/codec"
	"lark_rpc_v2/internal/limiter"
	"lark_rpc_v2/internal/protocol"
	"lark_rpc_v2/internal/transport"
)

type Server struct {
	addr     string
	services map[string]any
	limiter  *limiter.TokenBucket
	listener net.Listener
	handler  *Handler
	codec    codec.Codec

	conns   map[*transport.TCPConnection]struct{}
	closing chan struct{}
}

// Using another Go convention to create objects here
func mustNewHandler() *Handler {
	h, err := NewHandler(nil, WithHandlerCodec(codec.JSON))
	if err != nil {
		panic(err)
	}
	return h
}

func NewServer(addr string, opts ...ServerOption) (*Server, error) {
	s := &Server{
		addr:     addr,
		services: make(map[string]any),
		limiter:  limiter.NewTokenBucket(10000),
		handler:  mustNewHandler(),
		conns:    make(map[*transport.TCPConnection]struct{}),
		closing:  make(chan struct{}),
	}

	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Server) Register(name string, service any) {
	s.services[name] = service
}

// Single-connection single-goroutine serial model, requests can reuse connection like HTTP/1.1
// But currently at response level, it sequentially executes the first request, then the second
// TODO: Optimize with streams later
func (s *Server) Handle(conn *transport.TCPConnection) {
	defer conn.Close()
	log.Println("Test once")
	for {
		// Read request
		msg, err := conn.Read()
		if err != nil {
			// Connection closed or error, exit
			return
		}

		// Rate limit check
		if !s.limiter.Allow() {
			resp := &protocol.Message{
				Header: &protocol.Header{
					RequestID:   msg.Header.RequestID,
					Error:       "rate limit exceeded",
					Compression: codec.CompressionGzip,
				},
			}
			conn.Write(resp)
			continue
		}
		// Handle request
		s.handler.Process(conn, msg, s.services[msg.Header.ServiceName])
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listener = ln

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-s.closing:
				return nil
			default:
				continue
			}
		}

		tcpConn := transport.NewTCPConnection(conn)

		s.conns[tcpConn] = struct{}{}

		go func() {
			s.Handle(tcpConn)
			delete(s.conns, tcpConn)
		}()
	}

}

func (s *Server) Close() {
	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *Server) Shutdown() {
	close(s.closing)

	if s.listener != nil {
		s.listener.Close()
	}

	for conn := range s.conns {
		conn.Close()
	}

	log.Println("server shutdown complete")
}
