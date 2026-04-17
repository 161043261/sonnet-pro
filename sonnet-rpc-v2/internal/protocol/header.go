package protocol

import "lark_rpc_v2/internal/codec"

// CodecType codec type
type CodecType byte

const (
	CodecTypeJSON CodecType = iota + 1
	CodecTypeProto
)

type Header struct {
	RequestID   uint64
	ServiceName string
	MethodName  string
	Error       string
	CodecType   CodecType
	Compression codec.CompressionType
}
