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

func benchmarkSimpleNumberWrite(b *testing.B, n int64, fn func(*resp.Writer, int64) (int, error)) {
	w := resp.NewWriter(ioutil.Discard)

	for i := 0; i < b.N; i++ {
		if _, err := fn(w, n); err != nil {
			b.Fatalf("write failed: %s", err)
		}
	}
}

func benchmarkSimpleWrite(b *testing.B, s string, fn func(*resp.Writer, string) (int, error)) {
	w := resp.NewWriter(ioutil.Discard)

	for i := 0; i < b.N; i++ {
		if _, err := fn(w, s); err != nil {
			b.Fatalf("write failed: %s", err)
		}
	}
}

type simpleWriteCase struct {
	Name     string
	Expected string
	In       []byte
}

func prefixedSimpleWriteCases(prefix string) []simpleWriteCase {
	return []simpleWriteCase{
		{
			Name:     "empty",
			Expected: prefix + "\r\n",
			In:       []byte{},
		},
		{
			Name:     "nil",
			Expected: prefix + "\r\n",
			In:       nil,
		},
		{
			Name:     "small",
			Expected: prefix + "YO hello world\r\n",
			In:       []byte("YO hello world"),
		},
		{
			Name:     "invalid",
			Expected: prefix + "YO hello\r\nworld\r\n",
			In:       []byte("YO hello\r\nworld"),
		},
	}
}

func (s simpleWriteCase) run(t *testing.T,
	stringsFunc func(*resp.Writer, string) (int, error),
	bytesFunc func(*resp.Writer, []byte) (int, error)) {

	t.Run(s.Name, func(t *testing.T) {
		s.runBytes(t, bytesFunc)

		if s.In != nil {
			s.runString(t, stringsFunc)
		}
	})
}

func (s simpleWriteCase) runBytes(t *testing.T, fn func(*resp.Writer, []byte) (int, error)) {
	t.Run("Bytes", func(t *testing.T) {
		var buf bytes.Buffer
		w := resp.NewWriter(&buf)

		if _, err := fn(w, s.In); err != nil {
			t.Errorf("write failed: %s", err)
		} else if got := buf.String(); got != s.Expected {
			t.Errorf("got %q, expected %q", got, s.Expected)
		}
	})
}

func (s simpleWriteCase) runString(t *testing.T, fn func(*resp.Writer, string) (int, error)) {
	t.Run("String", func(t *testing.T) {
		var buf bytes.Buffer
		w := resp.NewWriter(&buf)

		if _, err := fn(w, string(s.In)); err != nil {
			t.Errorf("write failed: %s", err)
		} else if got := buf.String(); got != s.Expected {
			t.Errorf("got %q, expected %q", got, s.Expected)
		}
	})
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
				t.Errorf("comparison failed. got %q, expected %q", got, test.Data)
			}
		})
	}
}

func TestWriterWriteArrayHeader(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Expected string
		Err      error
		N        int64
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
	for _, n := range []int64{-1, 0, 100} {
		b.Run(strconv.FormatInt(n, 10), func(b *testing.B) {
			benchmarkSimpleNumberWrite(b, n, (*resp.Writer).WriteArrayHeader)
		})
	}
}

func TestWriterWriteBlobString(t *testing.T) {
	for _, test := range []simpleWriteCase{
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
		test.run(t,
			(*resp.Writer).WriteBlobString,
			(*resp.Writer).WriteBlobStringBytes)
	}
}

func BenchmarkWriterWriteBlobString(b *testing.B) {
	for _, n := range []int64{-1, 0, 100} {
		b.Run(strconv.FormatInt(n, 10), func(b *testing.B) {
			benchmarkSimpleNumberWrite(b, n, (*resp.Writer).WriteBlobStringHeader)
		})
	}
}

func TestWriterWriteBlobStringHeader(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Expected string
		Err      error
		N        int64
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
			Err:  resp.ErrInvalidBlobStringLength,
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

			if _, err := w.WriteBlobStringHeader(test.N); err != test.Err {
				t.Errorf("got error %v, expected %v", err, test.Err)
			} else if got := buf.String(); got != test.Expected {
				t.Errorf("got %q, expected %q", got, test.Expected)
			}
		})
	}
}

func BenchmarkWriterWriteBlobStringHeader(b *testing.B) {
	for _, n := range []int{0, 1, 10, 100, 1000, 10000} {
		s := strings.Repeat("X", n)

		b.Run(strconv.Itoa(n), func(b *testing.B) {
			benchmarkSimpleWrite(b, s, (*resp.Writer).WriteBlobString)
		})
	}
}

func TestWriterWriteSimpleError(t *testing.T) {
	for _, test := range prefixedSimpleWriteCases("-") {
		test.run(t,
			(*resp.Writer).WriteSimpleError,
			(*resp.Writer).WriteSimpleErrorBytes)
	}
}

func BenchmarkWriterWriteSimpleError(b *testing.B) {
	for _, n := range []int{0, 1, 10, 100, 1000, 10000} {
		s := strings.Repeat("X", n)

		b.Run(strconv.Itoa(n), func(b *testing.B) {
			benchmarkSimpleWrite(b, s, (*resp.Writer).WriteSimpleError)
		})
	}
}

func TestWriterWriteNumber(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Expected string
		I        int64
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

			if _, err := w.WriteNumber(test.I); err != nil {
				t.Errorf("got error %q", err)
			} else if got := buf.String(); got != test.Expected {
				t.Errorf("got %q, expected %q", got, test.Expected)
			}
		})
	}
}

func BenchmarkWriterWriteNumber(b *testing.B) {
	for _, n := range []int64{-1, 0, 100} {
		b.Run(strconv.FormatInt(n, 10), func(b *testing.B) {
			benchmarkSimpleNumberWrite(b, n, (*resp.Writer).WriteNumber)
		})
	}
}

func TestWriterWriteSimpleString(t *testing.T) {
	for _, test := range prefixedSimpleWriteCases("+") {
		test.run(t,
			(*resp.Writer).WriteSimpleString,
			(*resp.Writer).WriteSimpleStringBytes)
	}
}

func BenchmarkWriterWriteSimpleString(b *testing.B) {
	for _, n := range []int{0, 1, 10, 100, 1000, 10000} {
		s := strings.Repeat("X", n)

		b.Run(strconv.Itoa(n), func(b *testing.B) {
			benchmarkSimpleWrite(b, s, (*resp.Writer).WriteSimpleString)
		})
	}
}
