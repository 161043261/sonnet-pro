package loadbalance

import "lark_rpc_v2/internal/registry"

type LoadBalancer interface {
	Select([]registry.Instance) registry.Instance
}
