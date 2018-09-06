// +build integration

package resp_test

import (
	"io"
	"net"
	"os"
	"strings"
	"testing"
)

const (
	defaultRedisHost = "127.0.0.1:6379"
)

func dialRedis(tb testing.TB) io.ReadWriteCloser {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = defaultRedisHost
	}

	proto := "tcp"
	if strings.HasPrefix(host, "/") {
		proto = "unix"
	}

	conn, err := net.Dial(proto, host)
	if err != nil {
		tb.Fatalf("failed to dial redis: %s", err)
	}

	return conn
}

func withRedisConn(tb testing.TB, f func(io.ReadWriteCloser)) {
	conn := dialRedis(tb)
	defer func() {
		if err := conn.Close(); err != nil {
			tb.Errorf("failed to close connection to redis: %s", err)
		}
	}()

	if _, err := conn.Write([]byte("*1\r\n$8\r\nFLUSHALL\r\n")); err != nil {
		tb.Fatalf("failed to flush redis: %s", err)
	}

	respBuf := make([]byte, len("+OK\r\n"))
	if _, err := io.ReadFull(conn, respBuf); err != nil || string(respBuf) != "+OK\r\n" {
		tb.Fatalf("failed to flush redis: %s", respBuf)
	}

	f(conn)
}
