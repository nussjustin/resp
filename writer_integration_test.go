// +build integration

package resp_test

import (
	"bufio"
	"io"
	"testing"

	"github.com/nussjustin/resp"
)

func assertReadLines(tb testing.TB, br *bufio.Reader, expected ...string) {
	tb.Helper()

	for i := range expected {
		got, err := br.ReadSlice('\n')
		if err != nil {
			tb.Fatalf("failed to read line: %s", err)
		}

		if string(got) != expected[i]+"\r\n" {
			tb.Fatalf("got %q, expected %q", got, expected[i]+"\r\n")
		}
	}
}

func mustWriteBytesFunc(tb testing.TB, typeName string, f func([]byte) (int, error), s []byte) {
	tb.Helper()

	if _, err := f(s); err != nil {
		tb.Fatalf("failed to write %s: %s", typeName, err)
	}
}

func mustWriteIntegerFunc(tb testing.TB, typeName string, f func(int) (int, error), n int) {
	tb.Helper()

	if _, err := f(n); err != nil {
		tb.Fatalf("failed to write %s: %s", typeName, err)
	}
}

func mustWriteArrayHeader(tb testing.TB, w *resp.Writer, n int) {
	tb.Helper()
	mustWriteIntegerFunc(tb, "array header", w.WriteArrayHeader, n)
}

func mustWriteBulkString(tb testing.TB, w *resp.Writer, s []byte) {
	tb.Helper()
	mustWriteBytesFunc(tb, "bulk string", w.WriteBulkString, s)
}

func TestWriterIntegration(t *testing.T) {
	withRedisConn(t, func(conn io.ReadWriteCloser) {
		br := bufio.NewReader(conn)
		w := resp.NewWriter(conn)

		mustWriteArrayHeader(t, w, 3)
		mustWriteBulkString(t, w, []byte("SET"))
		mustWriteBulkString(t, w, []byte("hello"))
		mustWriteBulkString(t, w, []byte("world"))
		assertReadLines(t, br, "+OK")

		mustWriteArrayHeader(t, w, 4)
		mustWriteBulkString(t, w, []byte("SET"))
		mustWriteBulkString(t, w, []byte("hello"))
		mustWriteBulkString(t, w, []byte("world!"))
		mustWriteBulkString(t, w, []byte("NX"))
		assertReadLines(t, br, "$-1")

		mustWriteArrayHeader(t, w, 2)
		mustWriteBulkString(t, w, []byte("GET"))
		mustWriteBulkString(t, w, []byte("hello"))
		assertReadLines(t, br, "$5", "world")

		mustWriteArrayHeader(t, w, 4)
		mustWriteBulkString(t, w, []byte("SADD"))
		mustWriteBulkString(t, w, []byte("set1"))
		mustWriteBulkString(t, w, []byte("foo"))
		mustWriteBulkString(t, w, []byte("bar"))
		assertReadLines(t, br, ":2")

		mustWriteArrayHeader(t, w, 3)
		mustWriteBulkString(t, w, []byte("SADD"))
		mustWriteBulkString(t, w, []byte("set1"))
		mustWriteBulkString(t, w, []byte("baz"))
		assertReadLines(t, br, ":1")

		mustWriteArrayHeader(t, w, 3)
		mustWriteBulkString(t, w, []byte("SADD"))
		mustWriteBulkString(t, w, []byte("set1"))
		mustWriteBulkString(t, w, []byte("baz"))
		assertReadLines(t, br, ":0")

		mustWriteArrayHeader(t, w, 5)
		mustWriteBulkString(t, w, []byte("SREM"))
		mustWriteBulkString(t, w, []byte("set1"))
		mustWriteBulkString(t, w, []byte("foo"))
		mustWriteBulkString(t, w, []byte("baz"))
		mustWriteBulkString(t, w, []byte("qux"))
		assertReadLines(t, br, ":2")

		mustWriteArrayHeader(t, w, 2)
		mustWriteBulkString(t, w, []byte("SMEMBERS"))
		mustWriteBulkString(t, w, []byte("set1"))
		assertReadLines(t, br, "*1", "$3", "bar")

		mustWriteArrayHeader(t, w, 2)
		mustWriteBulkString(t, w, []byte("ZCARD"))
		mustWriteBulkString(t, w, []byte("set1"))
		assertReadLines(t, br, "-WRONGTYPE Operation against a key holding the wrong kind of value")
	})
}
