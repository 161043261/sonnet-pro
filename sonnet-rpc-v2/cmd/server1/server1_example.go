package main

import (
	"log"
	"os"
	"os/signal"
	"lark_rpc_v2/internal/codec"
	"lark_rpc_v2/internal/registry"
	"lark_rpc_v2/internal/server"
	"lark_rpc_v2/pkg/api"
)

func main() {
	reg, err := registry.NewRegistry([]string{"localhost:2379"})
	if err != nil {
		log.Fatal(err)
	}

	srv, err := server.NewServer(":9090", server.WithServerCodec(codec.JSON))
	if err != nil {
		log.Println("server.NewServer error ", err.Error())
		return
	}
	// Register Arith service
	srv.Register("Arith", &api.Arith{})
	srv.Register("Arith2", &api.Arith2{})
	// Register service to etcd
	err = reg.Register("Arith", registry.Instance{
		Addr: "localhost:9090",
	}, 10)
	if err != nil {
		log.Fatal(err)
	}
	err = reg.Register("Arith2", registry.Instance{
		Addr: "localhost:9090",
	}, 10)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("server started at :9090")
	srv.Start()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	go func() {
		<-sigCh
		log.Println("graceful shutdown...")
		srv.Shutdown()
	}()

}
