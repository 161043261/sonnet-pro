package lark_cache_v2

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"crypto/tls"

	pb "github.com/hangtiancheng/lark_cache_v2/pb"
	"github.com/hangtiancheng/lark_cache_v2/registry"
	"github.com/sirupsen/logrus"
	client_v3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Server defines cache server
type Server struct {
	pb.UnimplementedlarkCacheServer
	addr       string            // Service address
	svcName    string            // Service name
	groups     *sync.Map         // Cache groups
	grpcServer *grpc.Server      // gRPC server
	etcdCli    *client_v3.Client // etcd client
	stopCh     chan error        // Stop signal
	opts       *ServerOptions    // Server options
}

// ServerOptions server configuration options
type ServerOptions struct {
	EtcdEndpoints []string      // etcd endpoints
	DialTimeout   time.Duration // Dial timeout
	MaxMsgSize    int           // Max message size
	TLS           bool          // Enable TLS
	CertFile      string        // Certificate file
	KeyFile       string        // Key file
}

// DefaultServerOptions default configuration
var DefaultServerOptions = &ServerOptions{
	EtcdEndpoints: []string{"localhost:2379"},
	DialTimeout:   5 * time.Second,
	MaxMsgSize:    4 << 20, // 4MB
}

// ServerOption defines option function type
type ServerOption func(*ServerOptions)

// WithEtcdEndpoints sets etcd endpoints
func WithEtcdEndpoints(endpoints []string) ServerOption {
	return func(o *ServerOptions) {
		o.EtcdEndpoints = endpoints
	}
}

// WithDialTimeout sets dial timeout
func WithDialTimeout(timeout time.Duration) ServerOption {
	return func(o *ServerOptions) {
		o.DialTimeout = timeout
	}
}

// WithTLS sets TLS configuration
func WithTLS(certFile, keyFile string) ServerOption {
	return func(o *ServerOptions) {
		o.TLS = true
		o.CertFile = certFile
		o.KeyFile = keyFile
	}
}

// NewServer creates new server instance
func NewServer(addr, svcName string, opts ...ServerOption) (*Server, error) {
	options := DefaultServerOptions
	for _, opt := range opts {
		opt(options)
	}

	// Create etcd client
	etcdCli, err := client_v3.New(client_v3.Config{
		Endpoints:   options.EtcdEndpoints,
		DialTimeout: options.DialTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %v", err)
	}

	// Create gRPC server
	var serverOpts []grpc.ServerOption
	serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(options.MaxMsgSize))

	if options.TLS {
		creds, err := loadTLSCredentials(options.CertFile, options.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS credentials: %v", err)
		}
		serverOpts = append(serverOpts, grpc.Creds(creds))
	}

	srv := &Server{
		addr:       addr,
		svcName:    svcName,
		groups:     &sync.Map{},
		grpcServer: grpc.NewServer(serverOpts...),
		etcdCli:    etcdCli,
		stopCh:     make(chan error),
		opts:       options,
	}

	// Register service
	pb.RegisterlarkCacheServer(srv.grpcServer, srv)

	// Register health check service
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(srv.grpcServer, healthServer)
	healthServer.SetServingStatus(svcName, healthpb.HealthCheckResponse_SERVING)

	return srv, nil
}

// Start starts server
func (s *Server) Start() error {
	// Start gRPC server
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	// Register to etcd
	stopCh := make(chan error)
	go func() {
		if err := registry.Register(s.svcName, s.addr, stopCh); err != nil {
			logrus.Errorf("failed to register service: %v", err)
			close(stopCh)
			return
		}
	}()

	logrus.Infof("Server starting at %s", s.addr)
	return s.grpcServer.Serve(lis)
}

// Stop stops server
func (s *Server) Stop() {
	close(s.stopCh)
	s.grpcServer.GracefulStop()
	if s.etcdCli != nil {
		s.etcdCli.Close()
	}
}

// Get implements Get method of Cache service
func (s *Server) Get(ctx context.Context, req *pb.Request) (*pb.ResponseForGet, error) {
	group := GetGroup(req.Group)
	if group == nil {
		return nil, fmt.Errorf("group %s not found", req.Group)
	}

	view, err := group.Get(ctx, req.Key)
	if err != nil {
		return nil, err
	}

	return &pb.ResponseForGet{Value: view.ByteSLice()}, nil
}

// Set implements Set method of Cache service
func (s *Server) Set(ctx context.Context, req *pb.Request) (*pb.ResponseForGet, error) {
	group := GetGroup(req.Group)
	if group == nil {
		return nil, fmt.Errorf("group %s not found", req.Group)
	}

	// Get flag from context, create new context if not exists
	fromPeer := ctx.Value("from_peer")
	if fromPeer == nil {
		ctx = context.WithValue(ctx, "from_peer", true)
	}

	if err := group.Set(ctx, req.Key, req.Value); err != nil {
		return nil, err
	}

	return &pb.ResponseForGet{Value: req.Value}, nil
}

// Delete implements Delete method of Cache service
func (s *Server) Delete(ctx context.Context, req *pb.Request) (*pb.ResponseForDelete, error) {
	group := GetGroup(req.Group)
	if group == nil {
		return nil, fmt.Errorf("group %s not found", req.Group)
	}

	err := group.Delete(ctx, req.Key)
	return &pb.ResponseForDelete{Value: err == nil}, err
}

// loadTLSCredentials loads TLS credentials
func loadTLSCredentials(certFile, keyFile string) (credentials.TransportCredentials, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
	}), nil
}
