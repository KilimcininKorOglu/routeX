package dnsMITMProxy

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type plainUpstream struct {
	tcpPool    *connPool
	udpPool    *connPool
	bufferPool *sync.Pool
	timeout    time.Duration
}

func newPlainUpstream(addr string, maxIdleConns uint, timeout time.Duration) *plainUpstream {
	return &plainUpstream{
		tcpPool: newConnPool("tcp", addr, maxIdleConns),
		udpPool: newConnPool("udp", addr, maxIdleConns),
		bufferPool: &sync.Pool{
			New: func() interface{} {
				buf := make([]byte, dns.MaxMsgSize)
				return &buf
			},
		},
		timeout: timeout,
	}
}

func (u *plainUpstream) Query(ctx context.Context, req []byte, network string) ([]byte, error) {
	var pool *connPool
	if network == "tcp" {
		pool = u.tcpPool
	} else {
		pool = u.udpPool
	}

	conn, err := pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to upstream DNS server: %w", err)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(u.timeout)
	}
	if err := conn.SetDeadline(deadline); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to set deadline: %w", err)
	}

	if network == "tcp" {
		lenBuf := []byte{byte(len(req) >> 8), byte(len(req))}
		if _, err := conn.Write(lenBuf); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("failed to write length: %w", err)
		}
	}

	if _, err := conn.Write(req); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	var resp []byte
	if network == "tcp" {
		lenBuf := make([]byte, 2)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("failed to read length: %w", err)
		}
		respLen := int(lenBuf[0])<<8 | int(lenBuf[1])
		if respLen > maxTCPMsgSize {
			_ = conn.Close()
			return nil, fmt.Errorf("response too large: %d", respLen)
		}
		resp = make([]byte, respLen)
		if _, err := io.ReadFull(conn, resp); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
	} else {
		bufPtr := u.bufferPool.Get().(*[]byte)
		defer u.bufferPool.Put(bufPtr)
		buf := *bufPtr
		n, err := conn.Read(buf)
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		resp = make([]byte, n)
		copy(resp, buf[:n])
	}

	pool.Put(conn)
	return resp, nil
}

func (u *plainUpstream) Close() error {
	u.tcpPool.Close()
	u.udpPool.Close()
	return nil
}
