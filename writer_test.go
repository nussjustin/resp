package resp_test

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"github.com/nussjustin/resp"
)

func assertBytes(tb testing.TB, got []byte, expected string) {
	tb.Helper()

	if gotstr := string(got); gotstr != expected {
		tb.Errorf("read failed. got %q, expected %q", gotstr, expected)
	}
}

func mustWrite(tb testing.TB, w io.Writer, b []byte) {
	tb.Helper()

	if n, err := w.Write(b); err != nil {
		tb.Fatalf("write failed: %s", err)
	} else if n < len(b) {
		tb.Fatalf("failed to write all bytes. wrote %d, expected %d", n, len(b))
	}
}

func TestWriterReset(t *testing.T) {
	var b1 bytes.Buffer
	bw1 := bufio.NewWriter(&b1)
	w := resp.NewWriter(bw1)

	mustWrite(t, w, []byte("hello"))
	bw1.Flush()
	assertBytes(t, b1.Bytes(), "hello")

	var b2 bytes.Buffer
	bw2 := bufio.NewWriter(&b2)
	w.Reset(bw2)

	mustWrite(t, w, []byte("world"))
	bw1.Flush()
	bw2.Flush()
	assertBytes(t, b1.Bytes(), "hello")
	assertBytes(t, b2.Bytes(), "world")

	var b3 bytes.Buffer
	w.Reset(&b3)
	mustWrite(t, w, []byte("!"))
	bw1.Flush()
	bw2.Flush()
	assertBytes(t, b1.Bytes(), "hello")
	assertBytes(t, b2.Bytes(), "world")
	assertBytes(t, b3.Bytes(), "!")
}

func benchmarkSimpleIntegerWrite(b *testing.B, n int, fn func(*resp.Writer, int) (int, error)) {
	w := resp.NewWriter(ioutil.Discard)

	for i := 0; i < b.N; i++ {
		if _, err := fn(w, n); err != nil {
			b.Fatalf("write failed: %s", err)
		}
	}
}

func benchmarkSimpleWrite(b *testing.B, s []byte, fn func(*resp.Writer, []byte) (int, error)) {
	w := resp.NewWriter(ioutil.Discard)

	for i := 0; i < b.N; i++ {
		if _, err := fn(w, s); err != nil {
			b.Fatalf("write failed: %s", err)
		}
	}
}

func testSimpleWrite(tb testing.TB, input, expected []byte, fn func(*resp.Writer, []byte) (int, error)) {
	tb.Helper()

	var buf bytes.Buffer
	w := resp.NewWriter(&buf)

	if _, err := fn(w, input); err != nil {
		tb.Errorf("write failed: %s", err)
	} else if got := buf.Bytes(); !bytes.Equal(got, expected) {
		tb.Errorf("got %q, expected %q", got, expected)
	}
}

func TestWriterWrite(t *testing.T) {
	for _, test := range []struct {
		Name string
		Data []byte
	}{
		{
			Name: "resp",
			Data: []byte("*-1\r\n"),
		},
		{
			Name: "invalid resp",
			Data: []byte("hello world"),
		},
		{
			Name: "empty",
			Data: []byte{},
		},
		{
			Name: "zero",
			Data: []byte{},
		},
		{
			Name: "nil",
			Data: nil,
		},
	} {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			var buf bytes.Buffer
			w := resp.NewWriter(&buf)
			mustWrite(t, w, test.Data)

			if got := buf.Bytes(); !bytes.Equal(got, test.Data) {
				t.Errorf("comparsion failed. got %q, expected %q", got, test.Data)
			}
		})
	}
}

func TestWriterWriteArrayHeader(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Expected string
		Err      error
		N        int
	}{
		{
			Name:     "nil",
			Expected: "*-1\r\n",
			N:        -1,
		},
		{
			Name:     "zero",
			Expected: "*0\r\n",
			N:        0,
		},
		{
			Name: "below -1",
			Err:  resp.ErrInvalidArrayLength,
			N:    -5,
		},
		{
			Name:     "one",
			Expected: "*1\r\n",
			N:        1,
		},
		{
			Name:     "two",
			Expected: "*2\r\n",
			N:        2,
		},
		{
			Name:     "big",
			Expected: "*123\r\n",
			N:        123,
		},
	} {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			var buf bytes.Buffer
			w := resp.NewWriter(&buf)

			if _, err := w.WriteArrayHeader(test.N); err != test.Err {
				t.Errorf("got error %v, expected %v", err, test.Err)
			} else if got := buf.String(); got != test.Expected {
				t.Errorf("got %q, expected %q", got, test.Expected)
			}
		})
	}
}

func BenchmarkWriterWriteArrayHeader(b *testing.B) {
	for _, n := range []int{-1, 0, 100} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			benchmarkSimpleIntegerWrite(b, n, (*resp.Writer).WriteArrayHeader)
		})
	}
}

func TestWriterWriteBulkString(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Expected string
		In       []byte
	}{
		{
			Name:     "empty",
			Expected: "$0\r\n\r\n",
			In:       []byte{},
		},
		{
			Name:     "nil",
			Expected: "$-1\r\n",
			In:       nil,
		},
		{
			Name:     "small",
			Expected: "$12\r\nhello world!\r\n",
			In:       []byte("hello world!"),
		},
		{
			Name:     "large",
			Expected: "$1200\r\n" + strings.Repeat("hello world!", 100) + "\r\n",
			In:       []byte(strings.Repeat("hello world!", 100)),
		},
		{
			Name:     "with \\r",
			Expected: "$12\r\nhello\rworld!\r\n",
			In:       []byte("hello\rworld!"),
		},
		{
			Name:     "with \\r\\n",
			Expected: "$13\r\nhello\r\nworld!\r\n",
			In:       []byte("hello\r\nworld!"),
		},
		{
			Name:     "with \\n",
			Expected: "$12\r\nhello\nworld!\r\n",
			In:       []byte("hello\nworld!"),
		},
	} {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			testSimpleWrite(t, test.In, []byte(test.Expected), (*resp.Writer).WriteBulkString)
		})
	}
}

func BenchmarkWriterWriteBulkString(b *testing.B) {
	for _, n := range []int{-1, 0, 100} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			benchmarkSimpleIntegerWrite(b, n, (*resp.Writer).WriteBulkStringHeader)
		})
	}
}

func TestWriterWriteBulkStringHeader(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Expected string
		Err      error
		N        int
	}{
		{
			Name:     "nil",
			Expected: "$-1\r\n",
			N:        -1,
		},
		{
			Name:     "zero",
			Expected: "$0\r\n",
			N:        0,
		},
		{
			Name: "below -1",
			Err:  resp.ErrInvalidBulkStringLength,
			N:    -5,
		},
		{
			Name:     "one",
			Expected: "$1\r\n",
			N:        1,
		},
		{
			Name:     "two",
			Expected: "$2\r\n",
			N:        2,
		},
		{
			Name:     "big",
			Expected: "$123\r\n",
			N:        123,
		},
	} {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			var buf bytes.Buffer
			w := resp.NewWriter(&buf)

			if _, err := w.WriteBulkStringHeader(test.N); err != test.Err {
				t.Errorf("got error %v, expected %v", err, test.Err)
			} else if got := buf.String(); got != test.Expected {
				t.Errorf("got %q, expected %q", got, test.Expected)
			}
		})
	}
}

func BenchmarkWriterWriteBulkStringHeader(b *testing.B) {
	for _, n := range []int{0, 1, 10, 100, 1000, 10000} {
		s := bytes.Repeat([]byte{'X'}, n)

		b.Run(strconv.Itoa(n), func(b *testing.B) {
			benchmarkSimpleWrite(b, s, (*resp.Writer).WriteBulkString)
		})
	}
}

func TestWriterWriteError(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Expected string
		In       []byte
	}{
		{
			Name:     "empty",
			Expected: "-\r\n",
			In:       []byte{},
		},
		{
			Name:     "nil",
			Expected: "-\r\n",
			In:       nil,
		},
		{
			Name:     "small",
			Expected: "-ERR hello world\r\n",
			In:       []byte("ERR hello world"),
		},
		{
			Name:     "invalid",
			Expected: "-ERR hello\r\nworld\r\n",
			In:       []byte("ERR hello\r\nworld"),
		},
	} {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			testSimpleWrite(t, test.In, []byte(test.Expected), (*resp.Writer).WriteError)
		})
	}
}

func BenchmarkWriterWriteError(b *testing.B) {
	for _, n := range []int{0, 1, 10, 100, 1000, 10000} {
		s := bytes.Repeat([]byte{'X'}, n)

		b.Run(strconv.Itoa(n), func(b *testing.B) {
			benchmarkSimpleWrite(b, s, (*resp.Writer).WriteError)
		})
	}
}

func TestWriterWriteInteger(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Expected string
		I        int
	}{
		{
			Name:     "zero",
			Expected: ":0\r\n",
			I:        0,
		},
		{
			Name:     "small",
			Expected: ":2\r\n",
			I:        2,
		},
		{
			Name:     "small negative",
			Expected: ":-1\r\n",
			I:        -1,
		},
		{
			Name:     "large",
			Expected: ":6379\r\n",
			I:        6379,
		},
		{
			Name:     "large negative",
			Expected: ":-6379\r\n",
			I:        -6379,
		},
	} {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			var buf bytes.Buffer
			w := resp.NewWriter(&buf)

			if _, err := w.WriteInteger(test.I); err != nil {
				t.Errorf("got error %q", err)
			} else if got := buf.String(); got != test.Expected {
				t.Errorf("got %q, expected %q", got, test.Expected)
			}
		})
	}
}

func BenchmarkWriterWriteInteger(b *testing.B) {
	for _, n := range []int{-1, 0, 100} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			benchmarkSimpleIntegerWrite(b, n, (*resp.Writer).WriteInteger)
		})
	}
}

func TestWriterWriteSimpleString(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Expected string
		In       []byte
	}{
		{
			Name:     "empty",
			Expected: "+\r\n",
			In:       []byte{},
		},
		{
			Name:     "nil",
			Expected: "+\r\n",
			In:       nil,
		},
		{
			Name:     "small",
			Expected: "+OK hello world\r\n",
			In:       []byte("OK hello world"),
		},
		{
			Name:     "invalid",
			Expected: "+OK hello\r\nworld\r\n",
			In:       []byte("OK hello\r\nworld"),
		},
	} {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			testSimpleWrite(t, test.In, []byte(test.Expected), (*resp.Writer).WriteSimpleString)
		})
	}
}

func BenchmarkWriterWriteSimpleString(b *testing.B) {
	for _, n := range []int{0, 1, 10, 100, 1000, 10000} {
		s := bytes.Repeat([]byte{'X'}, n)

		b.Run(strconv.Itoa(n), func(b *testing.B) {
			benchmarkSimpleWrite(b, s, (*resp.Writer).WriteSimpleString)
		})
	}
}
