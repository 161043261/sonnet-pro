package registry

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	client_v3 "go.etcd.io/etcd/client/v3"
)

// Config defines etcd client configuration
type Config struct {
	Endpoints   []string      // Cluster addresses
	DialTimeout time.Duration // Dial timeout duration
}

// DefaultConfig provides default configuration
var DefaultConfig = &Config{
	Endpoints:   []string{"localhost:2379"},
	DialTimeout: 5 * time.Second,
}

// Register registers service to etcd
func Register(svcName, addr string, stopCh <-chan error) error {
	cli, err := client_v3.New(client_v3.Config{
		Endpoints:   DefaultConfig.Endpoints,
		DialTimeout: DefaultConfig.DialTimeout,
	})
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %v", err)
	}

	localIP, err := getLocalIP()
	if err != nil {
		cli.Close()
		return fmt.Errorf("failed to get local IP: %v", err)
	}
	if addr[0] == ':' {
		addr = fmt.Sprintf("%s%s", localIP, addr)
	}

	// Create lease
	lease, err := cli.Grant(context.Background(), 10) // Increase lease time to 10 seconds
	if err != nil {
		cli.Close()
		return fmt.Errorf("failed to create lease: %v", err)
	}

	// Register service using full key path
	key := fmt.Sprintf("/services/%s/%s", svcName, addr)
	_, err = cli.Put(context.Background(), key, addr, client_v3.WithLease(lease.ID))
	if err != nil {
		cli.Close()
		return fmt.Errorf("failed to put key-value to etcd: %v", err)
	}

	// Keep lease alive
	keepAliveCh, err := cli.KeepAlive(context.Background(), lease.ID)
	if err != nil {
		cli.Close()
		return fmt.Errorf("failed to keep lease alive: %v", err)
	}

	// Handle lease renewal and service revocation
	go func() {
		defer cli.Close()
		for {
			select {
			case <-stopCh:
				// Service revocation, revoke lease
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				cli.Revoke(ctx, lease.ID)
				cancel()
				return
			case resp, ok := <-keepAliveCh:
				if !ok {
					logrus.Warn("keep alive channel closed")
					return
				}
				logrus.Debugf("successfully renewed lease: %d", resp.ID)
			}
		}
	}()

	logrus.Infof("Service registered: %s at %s", svcName, addr)
	return nil
}

func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no valid local IP found")
}
