package resp_test

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"math"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/nussjustin/resp"
)

func TestReaderReset(t *testing.T) {
	var r *resp.Reader

	for _, s := range [...]string{
		"hello",
		"world",
		"!",
	} {
		// use TimeoutReader so a second read fails
		sr := iotest.TimeoutReader(strings.NewReader(s))

		if r == nil {
			r = resp.NewReader(sr)
		} else {
			r.Reset(sr)
		}

		got := make([]byte, len(s))

		if _, err := io.ReadFull(r, got); err != nil {
			t.Fatalf("string %q: read failed: %s", s, err)
		} else if string(got) != s {
			t.Fatalf("string %q: read %q", s, got)
		}
	}

	var buf1, buf2 bytes.Buffer
	br1, br2 := bufio.NewReader(&buf1), bufio.NewReader(&buf2)

	buf1.WriteString("hello")
	r = resp.NewReader(br1)
	b1, _ := ioutil.ReadAll(r)
	assertBytes(t, b1, "hello")

	buf1.WriteString("hello")
	buf2.WriteString("world")
	r.Reset(br2)
	b2, _ := ioutil.ReadAll(r)
	assertBytes(t, buf1.Bytes(), "hello")
	assertBytes(t, b2, "world")
}

func TestReaderRead(t *testing.T) {
	for _, test := range []struct {
		Name      string
		NewReader func() io.Reader
	}{
		{
			Name:      "empty",
			NewReader: func() io.Reader { return strings.NewReader("") },
		},
		{
			Name:      "small",
			NewReader: func() io.Reader { return strings.NewReader(strings.Repeat("a", 100)) },
		},
		{
			Name:      "large",
			NewReader: func() io.Reader { return strings.NewReader(strings.Repeat("a", 100000)) },
		},
		{
			Name:      "dataerr",
			NewReader: func() io.Reader { return iotest.DataErrReader(strings.NewReader("abc")) },
		},
	} {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			var got, expected bytes.Buffer
			_, gerr := got.ReadFrom(resp.NewReader(test.NewReader()))
			_, err := expected.ReadFrom(test.NewReader())

			if gerr != err {
				t.Errorf("got error %v, expected %v", gerr, err)
			} else if !bytes.Equal(got.Bytes(), expected.Bytes()) {
				t.Errorf("got %q (len %d), expected %q (len %d)", got, got.Len(), expected, expected.Len())
			}
		})
	}
}

func benchmarkSimpleNumberRead(b *testing.B, in string, fn func(*resp.Reader) (int64, error)) {
	sr := strings.NewReader(in)
	r := resp.NewReader(sr)

	for i := 0; i < b.N; i++ {
		sr.Reset(in)
		r.Reset(sr)

		if _, err := fn(r); err != nil {
			b.Fatalf("read failed: %s", err)
		}
	}
}

func benchmarkSimpleRead(b *testing.B, in string, fn func(*resp.Reader, []byte) ([]byte, error)) {
	sr := strings.NewReader(in)
	r := resp.NewReader(sr)

	buf := make([]byte, 0, len(in))

	for i := 0; i < b.N; i++ {
		sr.Reset(in)
		r.Reset(sr)

		if _, err := fn(r, buf); err != nil {
			b.Fatalf("read failed: %s", err)
		}
	}
}

func testSimpleNumberRead(tb testing.TB, input string, expected int64, err error, fn func(*resp.Reader) (int64, error)) {
	tb.Helper()

	r := resp.NewReader(strings.NewReader(input))

	if got, gerr := fn(r); gerr != err {
		tb.Errorf("got error %v, expected %v", gerr, err)
	} else if got != expected {
		tb.Errorf("got %d, expected %d", got, expected)
	}
}

func testSimpleRead(tb testing.TB,
	input string,
	expected []byte,
	err error,
	fn func(*resp.Reader, []byte) ([]byte, error)) {
	tb.Helper()

	r := resp.NewReader(strings.NewReader(input))

	var dst []byte
	got, gerr := fn(r, dst)

	if gerr != err {
		tb.Errorf("got error %v, expected %v", gerr, err)
	}

	if err != nil {
		return
	}

	if !bytes.Equal(got, expected) {
		tb.Errorf("got %q, expected %q", got, expected)
	} else if expected != nil && got == nil {
		tb.Errorf("got %#v, expected %#v", got, expected)
	} else if expected == nil && got != nil {
		tb.Errorf("got %#v, expected %#v", got, expected)
	}
}

func TestReaderReadArrayHeader(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Expected int64
		Err      error
		In       string
	}{
		{
			Name: "empty",
			Err:  io.EOF,
			In:   "",
		},
		{
			Name: "invalid type",
			Err:  resp.ErrUnexpectedType,
			In:   "A",
		},
		{
			Name: "wrong type",
			Err:  resp.ErrUnexpectedType,
			In:   "$",
		},
		{
			Name: "negative",
			Err:  resp.ErrInvalidArrayLength,
			In:   "*-2\r\n",
		},
		{
			Name: "null",
			Err:  resp.ErrInvalidArrayLength,
			In:   "*-1\r\n",
		},
		{
			Name:     "zero",
			Expected: 0,
			In:       "*0\r\n",
		},
		{
			Name:     "small",
			Expected: 10,
			In:       "*10\r\n",
		},
		{
			Name:     "large",
			Expected: 1000,
			In:       "*1000\r\n",
		},
		{
			Name: "no \\r",
			Err:  resp.ErrUnexpectedEOL,
			In:   "*5\n",
		},
		{
			Name: "no \\r\\n",
			Err:  io.EOF,
			In:   "*5",
		},
		{
			Name: "no \\n",
			Err:  resp.ErrUnexpectedEOL,
			In:   "*5\r",
		},
		{
			Name: "no number",
			Err:  resp.ErrInvalidArrayLength,
			In:   "*a\r\n",
		},
		{
			Name: "wrong \\n character",
			Err:  resp.ErrUnexpectedEOL,
			In:   "*0\ra",
		},
		{
			Name: "wrong \\r character",
			Err:  resp.ErrInvalidArrayLength,
			In:   "*0a\n",
		},
	} {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			testSimpleNumberRead(t, test.In, test.Expected, test.Err, (*resp.Reader).ReadArrayHeader)
		})
	}
}

func BenchmarkReaderReadArrayHeader(b *testing.B) {
	for _, s := range []string{
		"*0\r\n",
		"*1\r\n",
		"*100\r\n",
		"*10000\r\n",
	} {
		b.Run(s, func(b *testing.B) {
			benchmarkSimpleNumberRead(b, s, (*resp.Reader).ReadArrayHeader)
		})
	}
}

func TestReaderReadBlobString(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Expected []byte
		Err      error
		In       string
	}{
		{
			Name: "empty",
			Err:  io.EOF,
			In:   "",
		},
		{
			Name: "invalid type",
			Err:  resp.ErrUnexpectedType,
			In:   "A",
		},
		{
			Name: "wrong type",
			Err:  resp.ErrUnexpectedType,
			In:   "*",
		},
		{
			Name: "null",
			Err:  resp.ErrInvalidBlobStringLength,
			In:   "$-1\r\n",
		},
		{
			Name: "negative",
			Err:  resp.ErrInvalidBlobStringLength,
			In:   "$-2\r\n",
		},
		{
			Name:     "zero",
			Expected: []byte{},
			In:       "$0\r\n\r\n",
		},
		{
			Name:     "small",
			Expected: []byte("hello"),
			In:       "$5\r\nhello\r\n",
		},
		{
			Name:     "large",
			Expected: bytes.Repeat([]byte("hello"), 100),
			In:       "$500\r\n" + strings.Repeat("hello", 100) + "\r\n",
		},
		{
			Name:     "larger than buffer",
			Expected: bytes.Repeat([]byte("hello world"), 1000),
			In:       "$11000\r\n" + strings.Repeat("hello world", 1000) + "\r\n",
		},
		{
			Name: "no \\r",
			Err:  resp.ErrUnexpectedEOL,
			In:   "$0\r\n\n",
		},
		{
			Name: "no \\r\\n",
			Err:  resp.ErrUnexpectedEOL,
			In:   "$0\r\n",
		},
		{
			Name: "no \\n",
			Err:  resp.ErrUnexpectedEOL,
			In:   "$0\r",
		},
		{
			Name: "null, no \\r",
			Err:  resp.ErrUnexpectedEOL,
			In:   "$-1\n",
		},
		{
			Name: "null, no \\r\\n",
			Err:  io.EOF,
			In:   "$-1",
		},
		{
			Name: "null, no \\n",
			Err:  resp.ErrUnexpectedEOL,
			In:   "$-1\r",
		},
		{
			Name: "content too long",
			Err:  resp.ErrUnexpectedEOL,
			In:   "$5\r\nhello world\r\n",
		},
		{
			Name: "content too short",
			Err:  resp.ErrUnexpectedEOL,
			In:   "$11\r\nhello\r\n",
		},
	} {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			testSimpleRead(t, test.In, test.Expected, test.Err, (*resp.Reader).ReadBlobString)
		})
	}
}

func BenchmarkReaderReadBlobString(b *testing.B) {
	for _, test := range []struct {
		Name string
		In   string
	}{
		{
			Name: "empty",
			In:   "$0\r\n\r\n",
		},
		{
			Name: "small",
			In:   "$5\r\nhello\r\n",
		},
		{
			Name: "large",
			In:   "$100\r\n" + strings.Repeat("a", 100) + "\r\n",
		},
	} {
		b.Run(test.Name, func(b *testing.B) {
			benchmarkSimpleRead(b, test.In, (*resp.Reader).ReadBlobString)
		})
	}
}

func TestReaderReadBlobStringHeader(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Expected int64
		Err      error
		In       string
	}{
		{
			Name: "empty",
			Err:  io.EOF,
			In:   "",
		},
		{
			Name: "invalid type",
			Err:  resp.ErrUnexpectedType,
			In:   "A",
		},
		{
			Name: "wrong type",
			Err:  resp.ErrUnexpectedType,
			In:   "*",
		},
		{
			Name: "negative",
			Err:  resp.ErrInvalidBlobStringLength,
			In:   "$-2\r\n",
		},
		{
			Name: "null",
			Err:  resp.ErrInvalidBlobStringLength,
			In:   "$-1\r\n",
		},
		{
			Name:     "zero",
			Expected: 0,
			In:       "$0\r\n",
		},
		{
			Name:     "small",
			Expected: 10,
			In:       "$10\r\n",
		},
		{
			Name:     "large",
			Expected: 1000,
			In:       "$1000\r\n",
		},
		{
			Name: "no \\r",
			Err:  resp.ErrUnexpectedEOL,
			In:   "$5\n",
		},
		{
			Name: "no \\r\\n",
			Err:  io.EOF,
			In:   "$5",
		},
		{
			Name: "no \\n",
			Err:  resp.ErrUnexpectedEOL,
			In:   "$5\r",
		},
		{
			Name: "no number",
			Err:  resp.ErrInvalidBlobStringLength,
			In:   "$a\r\n",
		},
		{
			Name: "wrong \\n character",
			Err:  resp.ErrUnexpectedEOL,
			In:   "$0\ra",
		},
		{
			Name: "wrong \\r character",
			Err:  resp.ErrInvalidBlobStringLength,
			In:   "$0a\n",
		},
	} {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			testSimpleNumberRead(t, test.In, test.Expected, test.Err, (*resp.Reader).ReadBlobStringHeader)
		})
	}
}

func BenchmarkReaderReadBlobStringHeader(b *testing.B) {
	for _, s := range []string{
		"$0\r\n",
		"$1\r\n",
		"$100\r\n",
		"$10000\r\n",
	} {
		b.Run(s, func(b *testing.B) {
			benchmarkSimpleNumberRead(b, s, (*resp.Reader).ReadBlobStringHeader)
		})
	}
}

func TestReaderReadSimpleError(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Expected []byte
		Err      error
		In       string
	}{
		{
			Name: "empty",
			Err:  io.EOF,
			In:   "",
		},
		{
			Name: "invalid type",
			Err:  resp.ErrUnexpectedType,
			In:   "A",
		},
		{
			Name: "wrong type",
			Err:  resp.ErrUnexpectedType,
			In:   "*",
		},
		{
			Name:     "zero",
			Expected: []byte{},
			In:       "-\r\n",
		},
		{
			Name:     "small",
			Expected: []byte("ERR"),
			In:       "-ERR\r\n",
		},
		{
			Name:     "large",
			Expected: []byte("ERR " + strings.Repeat("a", 100)),
			In:       "-ERR " + strings.Repeat("a", 100) + "\r\n",
		},
		{
			Name:     "larger than buffer",
			Expected: []byte("ERR " + strings.Repeat("hello world", 1000)),
			In:       "-ERR " + strings.Repeat("hello world", 1000) + "\r\n",
		},
		{
			Name: "no \\r",
			Err:  resp.ErrUnexpectedEOL,
			In:   "-ERR\n",
		},
		{
			Name: "no \\r\\n",
			Err:  resp.ErrUnexpectedEOL,
			In:   "-ERR",
		},
		{
			Name: "no \\n",
			Err:  resp.ErrUnexpectedEOL,
			In:   "-ERR\r",
		},
	} {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			testSimpleRead(t, test.In, test.Expected, test.Err, (*resp.Reader).ReadSimpleError)
		})
	}
}

func BenchmarkReaderReadSimpleError(b *testing.B) {
	for _, s := range []string{
		"-\r\n",
		"-ERR\r\n",
		"-ERR some long error text\r\n",
	} {
		b.Run(s, func(b *testing.B) {
			benchmarkSimpleRead(b, s, (*resp.Reader).ReadSimpleError)
		})
	}
}

func TestReaderReadNumber(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Expected int64
		Err      error
		In       string
	}{
		{
			Name: "empty",
			Err:  io.EOF,
			In:   "",
		},
		{
			Name: "invalid type",
			Err:  resp.ErrUnexpectedType,
			In:   "A",
		},
		{
			Name: "wrong type",
			Err:  resp.ErrUnexpectedType,
			In:   "*",
		},
		{
			Name:     "negative",
			Expected: -2,
			In:       ":-2\r\n",
		},
		{
			Name:     "null",
			Expected: -1,
			In:       ":-1\r\n",
		},
		{
			Name:     "zero",
			Expected: 0,
			In:       ":0\r\n",
		},
		{
			Name:     "small",
			Expected: 10,
			In:       ":10\r\n",
		},
		{
			Name:     "large",
			Expected: 1000,
			In:       ":1000\r\n",
		},
		{
			Name: "no \\r",
			Err:  resp.ErrUnexpectedEOL,
			In:   ":5\n",
		},
		{
			Name: "no \\r\\n",
			Err:  io.EOF,
			In:   ":5",
		},
		{
			Name: "no \\n",
			Err:  resp.ErrUnexpectedEOL,
			In:   ":5\r",
		},
		{
			Name: "no number",
			Err:  resp.ErrInvalidNumber,
			In:   ":a\r\n",
		},
		{
			Name: "wrong \\n character",
			Err:  resp.ErrUnexpectedEOL,
			In:   ":0\ra",
		},
		{
			Name: "wrong \\r character",
			Err:  resp.ErrInvalidNumber,
			In:   ":0a\n",
		},
	} {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			testSimpleNumberRead(t, test.In, test.Expected, test.Err, (*resp.Reader).ReadNumber)
		})
	}
}

func BenchmarkReaderReadNumber(b *testing.B) {
	for _, s := range []string{
		":-100\r\n",
		":-1\r\n",
		":0\r\n",
		":1\r\n",
		":100\r\n",
		":10000\r\n",
	} {
		b.Run(s, func(b *testing.B) {
			benchmarkSimpleNumberRead(b, s, (*resp.Reader).ReadNumber)
		})
	}
}

func TestReaderReadSimpleString(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Expected []byte
		Err      error
		In       string
	}{
		{
			Name: "empty",
			Err:  io.EOF,
			In:   "",
		},
		{
			Name: "invalid type",
			Err:  resp.ErrUnexpectedType,
			In:   "A",
		},
		{
			Name: "wrong type",
			Err:  resp.ErrUnexpectedType,
			In:   "*",
		},
		{
			Name:     "zero",
			Expected: []byte{},
			In:       "+\r\n",
		},
		{
			Name:     "small",
			Expected: []byte("OK"),
			In:       "+OK\r\n",
		},
		{
			Name:     "large",
			Expected: []byte("OK " + strings.Repeat("a", 100)),
			In:       "+OK " + strings.Repeat("a", 100) + "\r\n",
		},
		{
			Name:     "larger than buffer",
			Expected: []byte("OK " + strings.Repeat("hello world", 1000)),
			In:       "+OK " + strings.Repeat("hello world", 1000) + "\r\n",
		},
		{
			Name: "no \\r",
			Err:  resp.ErrUnexpectedEOL,
			In:   "+OK\n",
		},
		{
			Name: "no \\r\\n",
			Err:  resp.ErrUnexpectedEOL,
			In:   "+OK",
		},
		{
			Name: "no \\n",
			Err:  resp.ErrUnexpectedEOL,
			In:   "+OK\r",
		},
	} {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			testSimpleRead(t, test.In, test.Expected, test.Err, (*resp.Reader).ReadSimpleString)
		})
	}
}

func BenchmarkReaderReadSimpleString(b *testing.B) {
	for _, s := range []string{
		"+\r\n",
		"+ERR\r\n",
		"+ERR some long error text\r\n",
	} {
		b.Run(s, func(b *testing.B) {
			benchmarkSimpleRead(b, s, (*resp.Reader).ReadSimpleString)
		})
	}
}

func TestReaderReadNull(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Err      error
		In string
	}{
		{
			Name: "empty",
			Err:  io.EOF,
		},
		{
			Name: "invalid type",
			Err:  resp.ErrUnexpectedType,
			In:   "A",
		},
		{
			Name: "wrong type",
			Err:  resp.ErrUnexpectedType,
			In:   "*",
		},
		{
			Name:     "normal",
			In:       "_\r\n",
		},
		{
			Name: "no \\r",
			Err:  resp.ErrUnexpectedEOL,
			In:   "_\n",
		},
		{
			Name: "no \\r\\n",
			Err:  resp.ErrUnexpectedEOL,
			In:   "_",
		},
		{
			Name: "no \\n",
			Err:  resp.ErrUnexpectedEOL,
			In:   "_\r",
		},
	} {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			r := resp.NewReader(strings.NewReader(test.In))

			if err := r.ReadNull(); err != test.Err {
				t.Errorf("got error %v, expected %v", err, test.Err)
			}
		})
	}
}

func BenchmarkReaderReadNull(b *testing.B) {
	const in = "_\r\n"

	sr := strings.NewReader(in)
	r := resp.NewReader(sr)

	for i := 0; i < b.N; i++ {
		sr.Reset(in)
		r.Reset(sr)

		if err := r.ReadNull(); err != nil {
			b.Fatalf("read failed: %s", err)
		}
	}
}

func TestReaderReadDouble(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Expected float64
		Err      error
		In       string
	}{
		{
			Name: "empty",
			Err:  io.EOF,
			In:   "",
		},
		{
			Name: "invalid type",
			Err:  resp.ErrUnexpectedType,
			In:   "A",
		},
		{
			Name: "wrong type",
			Err:  resp.ErrUnexpectedType,
			In:   "*",
		},
		{
			Name:     "negative",
			Expected: -123.456,
			In:       ",-123.456\r\n",
		},
		{
			Name:     "zero",
			Expected: 0.0,
			In:       ",0.0\r\n",
		},
		{
			Name:     "no point",
			Expected: 0.0,
			In:       ",0\r\n",
		},
		{
			Name:     "small",
			Expected: 1.0,
			In:       ",1.0\r\n",
		},
		{
			Name:     "large",
			Expected: 1000.0001,
			In:       ",1000.0001\r\n",
		},
		{
			Name:     "positive infinity",
			Expected: math.Inf(1),
			In:       ",inf\r\n",
		},
		{
			Name:     "negative infinity",
			Expected: math.Inf(-1),
			In:       ",-inf\r\n",
		},
		{
			Name:     "large",
			Expected: 1000.0001,
			In:       ",1000.0001\r\n",
		},
		{
			Name: "no \\r",
			Err:  resp.ErrUnexpectedEOL,
			In:   ",0\n",
		},
		{
			Name: "no \\r\\n",
			Err:  io.EOF,
			In:   ",0",
		},
		{
			Name: "no \\n",
			Err:  resp.ErrUnexpectedEOL,
			In:   ",0\r",
		},
		{
			Name: "no number",
			Err:  resp.ErrInvalidDouble,
			In:   ",a\r\n",
		},
	} {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			r := resp.NewReader(strings.NewReader(test.In))

			if got, gerr := r.ReadDouble(); gerr != test.Err {
				t.Errorf("got error %v, expected %v", gerr, test.Err)
			} else if got != test.Expected {
				t.Errorf("got %f, expected %f", got, test.Expected)
			}
		})
	}
}

func BenchmarkReaderReadDouble(b *testing.B) {
	for _, s := range []string{
		",-10000.00001\r\n",
		",-100.001\r\n",
		",-1.1\r\n",
		",0\r\n",
		",1.1\r\n",
		",100.001\r\n",
		",10000.00001\r\n",
	} {
		b.Run(s, func(b *testing.B) {
			sr := strings.NewReader(s)
			r := resp.NewReader(sr)

			for i := 0; i < b.N; i++ {
				sr.Reset(s)
				r.Reset(sr)

				if _, err := r.ReadDouble(); err != nil {
					b.Fatalf("read failed: %s", err)
				}
			}
		})
	}
}

func TestReaderReadMixed(t *testing.T) {
	const data = "+OK\r\n-ERR something went wrong\r\n$5\r\nhello\r\n*3\r\n$5\r\nworld\r\n:5\r\n*-1\r\n"

	r := resp.NewReader(strings.NewReader(data))

	if s, err := r.ReadSimpleString(nil); err != nil || string(s) != "OK" {
		t.Fatalf("failed to read simple string: %q %s", s, err)
	}

	if s, err := r.ReadSimpleError(nil); err != nil || string(s) != "ERR something went wrong" {
		t.Fatalf("failed to read error: %q %s", s, err)
	}

	if s, err := r.ReadBlobString(nil); err != nil || string(s) != "hello" {
		t.Fatalf("failed to read blob string: %q %s", s, err)
	}

	if n, err := r.ReadArrayHeader(); err != nil || n != 3 {
		t.Fatalf("failed to read array header: %s", err)
	}

	if s, err := r.ReadBlobString(nil); err != nil || string(s) != "world" {
		t.Fatalf("failed to read blob string: %s", err)
	}

	if n, err := r.ReadNumber(); err != nil || n != 5 {
		t.Fatalf("failed to read number: %s", err)
	}

	if _, err := r.ReadArrayHeader(); err != resp.ErrInvalidArrayLength {
		t.Fatalf("failed to read array header: %s", err)
	}
}
