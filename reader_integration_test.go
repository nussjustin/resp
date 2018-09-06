// +build integration

package resp_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/nussjustin/resp"
)

func mustWriteLines(tb testing.TB, w io.Writer, lines ...string) {
	tb.Helper()

	for _, line := range lines {
		if _, err := w.Write([]byte(line + "\r\n")); err != nil {
			tb.Fatalf("failed to write line %q: %s", line, err)
		}
	}
}

func assertReadBytesFunc(tb testing.TB, typeName string, f func([]byte) ([]byte, error), expected []byte) {
	tb.Helper()

	if got, err := f(nil); err != nil {
		tb.Fatalf("failed to read %s: %s", typeName, err)
	} else if !bytes.Equal(got, expected) {
		tb.Fatalf("got %q, expected %q", got, expected)
	} else if (got == nil && expected != nil) || (got != nil && expected == nil) {
		tb.Fatalf("got %#v, expected %#v", got, expected)
	}
}

func assertReadIntegerFunc(tb testing.TB, typeName string, f func() (int, error), expected int) {
	tb.Helper()

	if got, err := f(); err != nil {
		tb.Fatalf("failed to read %s: %s", typeName, err)
	} else if got != expected {
		tb.Fatalf("got %d, expected %d", got, expected)
	}
}

func assertReadArrayHeader(tb testing.TB, r *resp.Reader, n int) {
	tb.Helper()
	assertReadIntegerFunc(tb, "array header", r.ReadArrayHeader, n)
}

func assertReadBulkString(tb testing.TB, r *resp.Reader, s []byte) {
	tb.Helper()
	assertReadBytesFunc(tb, "bulk string", r.ReadBulkString, s)
}

func assertReadError(tb testing.TB, r *resp.Reader, s []byte) {
	tb.Helper()
	assertReadBytesFunc(tb, "error", r.ReadError, s)
}

func assertReadInteger(tb testing.TB, r *resp.Reader, n int) {
	tb.Helper()
	assertReadIntegerFunc(tb, "integer", r.ReadInteger, n)
}

func assertReadSimpleString(tb testing.TB, r *resp.Reader, s []byte) {
	tb.Helper()
	assertReadBytesFunc(tb, "simple string", r.ReadSimpleString, s)
}

func TestReaderIntegration(t *testing.T) {
	withRedisConn(t, func(conn io.ReadWriteCloser) {
		r := resp.NewReader(conn)

		mustWriteLines(t, conn, "*2", "$3", "GET", "$6", "string")
		assertReadBulkString(t, r, nil)

		mustWriteLines(t, conn, "*3", "$3", "SET", "$6", "string", "$6", "value1")
		assertReadSimpleString(t, r, []byte("OK"))
		mustWriteLines(t, conn, "*4", "$3", "SET", "$6", "string", "$6", "value2", "$2", "NX")
		assertReadBulkString(t, r, nil)
		mustWriteLines(t, conn, "*2", "$3", "GET", "$6", "string")
		assertReadBulkString(t, r, []byte("value1"))

		mustWriteLines(t, conn, "*2", "$8", "SMEMBERS", "$3", "set")
		assertReadArrayHeader(t, r, 0)
		mustWriteLines(t, conn, "*3", "$4", "SADD", "$3", "set", "$6", "value3")
		assertReadInteger(t, r, 1)
		mustWriteLines(t, conn, "*3", "$4", "SADD", "$3", "set", "$6", "value3")
		assertReadInteger(t, r, 0)
		mustWriteLines(t, conn, "*2", "$8", "SMEMBERS", "$3", "set")
		assertReadArrayHeader(t, r, 1)
		assertReadBulkString(t, r, []byte("value3"))

		mustWriteLines(t, conn, "*4", "$4", "ZADD", "$3", "set", "$3", "100", "$6", "value4")
		assertReadError(t, r, []byte("WRONGTYPE Operation against a key holding the wrong kind of value"))
	})
}
