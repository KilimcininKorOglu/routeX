package dnsMITMProxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"time"
)

type dotUpstream struct {
	pool    *connPool
	timeout time.Duration
}

func newDoTUpstream(addr string, maxIdleConns uint, timeout time.Duration, skipVerify bool, serverName string) *dotUpstream {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: skipVerify,
	}
	if serverName != "" {
		tlsCfg.ServerName = serverName
	}
	return &dotUpstream{
		pool:    newTLSConnPool(addr, maxIdleConns, tlsCfg),
		timeout: timeout,
	}
}

func (u *dotUpstream) Query(ctx context.Context, req []byte, _ string) ([]byte, error) {
	conn, err := u.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to DoT upstream: %w", err)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(u.timeout)
	}
	if err := conn.SetDeadline(deadline); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to set deadline: %w", err)
	}

	lenBuf := []byte{byte(len(req) >> 8), byte(len(req))}
	if _, err := conn.Write(lenBuf); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to write length: %w", err)
	}
	if _, err := conn.Write(req); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	respLenBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, respLenBuf); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to read length: %w", err)
	}
	respLen := int(respLenBuf[0])<<8 | int(respLenBuf[1])
	if respLen > maxTCPMsgSize {
		_ = conn.Close()
		return nil, fmt.Errorf("response too large: %d", respLen)
	}

	resp := make([]byte, respLen)
	if _, err := io.ReadFull(conn, resp); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	u.pool.Put(conn)
	return resp, nil
}

func (u *dotUpstream) Close() error {
	u.pool.Close()
	return nil
}
