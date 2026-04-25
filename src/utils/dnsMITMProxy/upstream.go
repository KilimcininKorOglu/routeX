package dnsMITMProxy

import "context"

type Upstream interface {
	Query(ctx context.Context, req []byte, network string) ([]byte, error)
	Close() error
}

type UpstreamConfig struct {
	Protocol      string
	Address       string
	URL           string
	MaxIdleConns  uint
	TLSSkipVerify bool
	TLSServerName string
	Timeout       uint
}
