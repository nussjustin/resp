package resp_test

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/nussjustin/resp"
)

func copyReaderToWriter(tb testing.TB, rw *resp.ReadWriter) {
	for {
		ty, err := rw.Peek()
		if err == io.EOF {
			break
		}
		if err != nil {
			tb.Fatalf("failed to peek at next type: %s", err)
		}

		switch ty {
		case resp.TypeArray:
			n, err := rw.ReadArrayHeader()
			if err != nil {
				tb.Fatalf("failed to read array header: %s", err)
			}
			if _, err := rw.WriteArrayHeader(n); err != nil {
				tb.Fatalf("failed to write array header for array of size %d: %s", n, err)
			}
		case resp.TypeBulkString:
			s, err := rw.ReadBulkString(nil)
			if err != nil {
				tb.Fatalf("failed to bulk string: %s", err)
			}
			if _, err := rw.WriteBulkString(s); err != nil {
				tb.Fatalf("failed to write bulk string %q: %s", s, err)
			}
		case resp.TypeError:
			s, err := rw.ReadError(nil)
			if err != nil {
				tb.Fatalf("failed to read error: %s", err)
			}
			if _, err := rw.WriteError(s); err != nil {
				tb.Fatalf("failed to write error %q: %s", s, err)
			}
		case resp.TypeInteger:
			n, err := rw.ReadInteger()
			if err != nil {
				tb.Fatalf("failed to read integer: %s", err)
			}
			if _, err := rw.WriteInteger(n); err != nil {
				tb.Fatalf("failed to write integer size %d: %s", n, err)
			}
		case resp.TypeSimpleString:
			s, err := rw.ReadSimpleString(nil)
			if err != nil {
				tb.Fatalf("failed to read simple string: %s", err)
			}
			if _, err := rw.WriteSimpleString(s); err != nil {
				tb.Fatalf("failed to write simple string %q: %s", s, err)
			}
		case resp.TypeInvalid:
			tb.Fatal("found invalid type")
		default:
			tb.Fatalf("found unknown type: %#v", ty)
		}
	}
}

func getTestFiles(tb testing.TB) []string {
	files, err := filepath.Glob(filepath.Join("testdata", tb.Name(), "*.resp"))
	if err != nil {
		tb.Fatalf("failed to glob testdata directory: %s", err)
	}
	if len(files) == 0 {
		tb.Fatalf("no test files found")
	}
	return files
}

type simpleReadWriter struct {
	io.Reader
	io.Writer
}

func TestTypeString(t *testing.T) {
	for ty := resp.Type(0); ty < ^resp.Type(0); ty++ {
		if ts := ty.String(); ts != fmt.Sprint(ty) {
			t.Fatalf("got %v, expected %v", ts, fmt.Sprint(ty))
		}
	}
}

func testReadWriterUsingFile(t *testing.T, fileName string) {
	file, err := os.Open(fileName)
	if err != nil {
		t.Fatalf("failed to read file %s: %s", fileName, err)
	}
	defer file.Close()

	var out bytes.Buffer
	inHash, outHash := sha1.New(), sha1.New()

	rw := resp.NewReadWriter(&simpleReadWriter{
		Reader: io.TeeReader(file, inHash),
		Writer: io.MultiWriter(&out, outHash),
	})

	copyReaderToWriter(t, rw)

	if inSum, outSum := inHash.Sum(nil), outHash.Sum(nil); !bytes.Equal(inSum, outSum) {
		t.Errorf("sha1 hashes differ: got %x, expected %x", outSum, inSum)
		t.Logf("output:\n%s\n", &out)
	}
}

func TestReadWriter(t *testing.T) {
	for _, file := range getTestFiles(t) {
		file := file

		testName := filepath.Base(file)
		testName = testName[:len(testName)-len(filepath.Ext(testName))]

		t.Run(testName, func(t *testing.T) {
			testReadWriterUsingFile(t, file)
		})
	}
}

func benchmarkReadWriterUsingFile(b *testing.B, fileName string) {
	fileBytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		b.Fatalf("failed to read file %s: %s", fileName, err)
	}

	fileBytesReader := bytes.NewReader(nil)
	srw := &simpleReadWriter{
		Reader: fileBytesReader,
		Writer: ioutil.Discard,
	}

	rw := resp.NewReadWriter(nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		fileBytesReader.Reset(fileBytes)
		rw.Reset(srw)

		copyReaderToWriter(b, rw)
	}
}

func BenchmarkReadWriter(b *testing.B) {
	for _, file := range getTestFiles(b) {
		file := file

		testName := filepath.Base(file)
		testName = testName[:len(testName)-len(filepath.Ext(testName))]

		b.Run(testName, func(b *testing.B) {
			benchmarkReadWriterUsingFile(b, file)
		})
	}
}
