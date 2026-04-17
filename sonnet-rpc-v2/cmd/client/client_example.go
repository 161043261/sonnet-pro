package main

import (
	"context"
	"fmt"
	"log"
	"lark_rpc_v2/internal/client"
	"lark_rpc_v2/internal/codec"
	"lark_rpc_v2/internal/registry"
	"lark_rpc_v2/internal/transport"
	"lark_rpc_v2/pkg/api"
	"time"
)

func main() {
	reg, _ := registry.NewRegistry([]string{"localhost:2379"})

	c, err := client.NewClient(
		reg,
		client.WithClientCodec(codec.JSON),
	)
	if err != nil {
		log.Println("NewClient error:", err)
		return
	}

	type call struct {
		future *transport.Future
		args   *api.Args
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Println("Start sending requests periodically...")

	counter := 0

	for {
		<-ticker.C

		fmt.Println("====== New round of requests ======")

		var calls []call

		// Send 3 requests per round
		for i := 0; i < 3; i++ {
			args := &api.Args{
				A: counter,
				B: counter,
			}
			counter++

			f, err := c.InvokeAsync(
				context.Background(),
				"Arith",
				"Add",
				args,
			)
			if err != nil {
				log.Println("send error:", err)
				continue
			}

			calls = append(calls, call{f, args})
		}

		// Wait for the 3 requests in this round to complete
		doneCh := make(chan call)
		log.Println("Need to wait for", len(calls), "tasks to complete")
		for _, item := range calls {
			go func(it call) {
				<-it.future.DoneChan()
				doneCh <- it
			}(item)
		}

		for i := 0; i < len(calls); i++ {
			item := <-doneCh

			reply := &api.Reply{}
			err := item.future.GetResult(reply)
			if err != nil {
				log.Println("get error:", err)
				continue
			}

			fmt.Printf("Add %v+%v result: %v\n",
				item.args.A,
				item.args.B,
				reply.Result)
		}
	}
}
