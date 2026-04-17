package codec

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"sync"
)

// CompressionType compression type
type CompressionType byte

const (
	CompressionNone CompressionType = iota
	CompressionGzip
)

// Compressor compressor interface
type compressor interface {
	compress([]byte) ([]byte, error)
	decompress([]byte) ([]byte, error)
}

// GzipCompressor gzip compressor
type GzipCompressor struct{}

func (g *GzipCompressor) compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g *GzipCompressor) decompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}

var (
	gzipCompressor = &GzipCompressor{}
	compressorMu   sync.RWMutex
	compressors    = make(map[CompressionType]compressor)
)

// RegisterCompressor registers compressor
func RegisterCompressor(t CompressionType, c compressor) {
	compressorMu.Lock()
	defer compressorMu.Unlock()
	compressors[t] = c
}

// GetCompressor gets compressor
func GetCompressor(t CompressionType) compressor {
	compressorMu.RLock()
	defer compressorMu.RUnlock()
	return compressors[t]
}

func init() {
	RegisterCompressor(CompressionGzip, gzipCompressor)
}

// Compress compresses with specified type
func Compress(data []byte, t CompressionType) ([]byte, error) {
	c := GetCompressor(t)
	if c == nil {
		return nil, errors.New("compressor not found")
	}
	return c.compress(data)
}

// Decompress decompresses with specified type
func Decompress(data []byte, t CompressionType) ([]byte, error) {
	c := GetCompressor(t)
	if c == nil {
		return nil, errors.New("compressor not found")
	}
	return c.decompress(data)
}
