package dnsMITMProxy

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"
)

// connPool is a simple connection pool based on a channel
type connPool struct {
	network   string
	addr      string
	pool      chan net.Conn
	maxIdle   uint
	mu        sync.Mutex
	closed    bool
	tlsConfig *tls.Config
}

func newConnPool(network, addr string, maxIdle uint) *connPool {
	return &connPool{
		network: network,
		addr:    addr,
		pool:    make(chan net.Conn, maxIdle),
		maxIdle: maxIdle,
	}
}

func newTLSConnPool(addr string, maxIdle uint, cfg *tls.Config) *connPool {
	return &connPool{
		network:   "tcp",
		addr:      addr,
		pool:      make(chan net.Conn, maxIdle),
		maxIdle:   maxIdle,
		tlsConfig: cfg,
	}
}

func (p *connPool) Get(ctx context.Context) (net.Conn, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, errors.New("pool is closed")
	}
	p.mu.Unlock()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	select {
	case conn := <-p.pool:
		return conn, nil
	default:
		if p.tlsConfig != nil {
			d := tls.Dialer{Config: p.tlsConfig}
			return d.DialContext(ctx, p.network, p.addr)
		}
		var d net.Dialer
		return d.DialContext(ctx, p.network, p.addr)
	}
}

func (p *connPool) Put(conn net.Conn) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		_ = conn.Close()
		return
	}
	p.mu.Unlock()

	select {
	case p.pool <- conn:
	default:
		_ = conn.Close()
	}
}

func (p *connPool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()

	close(p.pool)
	for conn := range p.pool {
		_ = conn.Close()
	}
}
