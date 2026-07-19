package app

import (
	"context"
	"testing"
)

type benchmarkProxyCaller struct{}

func (benchmarkProxyCaller) Call(context.Context, string, any, any) error { return nil }

func BenchmarkProxyInitialization(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		_ = NewProxyServer(benchmarkProxyCaller{})
	}
}
