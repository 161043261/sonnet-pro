package loadbalance

import (
	"lark_rpc_v2/internal/registry"
	"sync/atomic"
)

type RoundRobin struct {
	idx uint64
}

func NewRR() *RoundRobin {
	r := &RoundRobin{}
	return r
}

func (r *RoundRobin) Select(list []registry.Instance) registry.Instance {
	i := atomic.AddUint64(&r.idx, 1)
	return list[i%uint64(len(list))]
}
