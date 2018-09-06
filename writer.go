package resp

import (
	"io"
	"strconv"
)

// Writer wraps an io.Writer and provides methods for writing the RESP protocol.
type Writer struct {
	w   io.Writer
	buf []byte
}

// NewWriter returns a *Writer that uses the given io.Writer for writes.
func NewWriter(w io.Writer) *Writer {
	var rw Writer
	rw.Reset(w)
	return &rw
}

var _ io.Writer = (*Writer)(nil)

// Reset sets the underlying io.Writer to w and resets all internal state.
func (rw *Writer) Reset(w io.Writer) {
	rw.buf = rw.buf[:0]
	rw.w = w
}

func (rw *Writer) writeBytes(prefix byte, s []byte) (int, error) {
	rw.buf = rw.buf[:0]
	rw.buf = append(rw.buf, prefix)
	rw.buf = append(rw.buf, s...)
	rw.buf = append(rw.buf, '\r', '\n')

	return rw.w.Write(rw.buf)
}

func (rw *Writer) writeNumber(prefix byte, n int64) (int, error) {
	rw.buf = rw.buf[:0]
	rw.buf = append(rw.buf, prefix)
	rw.buf = strconv.AppendInt(rw.buf, n, 10)
	rw.buf = append(rw.buf, '\r', '\n')

	return rw.w.Write(rw.buf)
}

func (rw *Writer) writeString(prefix byte, s string) (int, error) {
	rw.buf = rw.buf[:0]
	rw.buf = append(rw.buf, prefix)
	rw.buf = append(rw.buf, s...)
	rw.buf = append(rw.buf, '\r', '\n')

	return rw.w.Write(rw.buf)
}

// Write allows writing raw data to the underlying io.Writer.
//
// It implements the io.Writer interface.
func (rw *Writer) Write(dst []byte) (int, error) {
	return rw.w.Write(dst)
}

var nilArrayHeaderBytes = []byte("*-1\r\n")

// WriteArrayHeader writes an array header for an array of length n.
//
// If n is < -1, ErrInvalidArrayLength is returned.
func (rw *Writer) WriteArrayHeader(n int) (int, error) {
	if n < -1 {
		return 0, ErrInvalidArrayLength
	}

	if n == -1 { // fast-path
		return rw.w.Write(nilArrayHeaderBytes)
	}

	return rw.writeNumber('*', int64(n))
}

var nilBulkStringHeaderBytes = []byte("$-1\r\n")

// WriteBulkStringHeader writes a bulk string header for an bulk string of length n.
//
// If n is < -1, ErrInvalidBulkStringLength is returned.
func (rw *Writer) WriteBulkStringHeader(n int) (int, error) {
	if n < -1 {
		return 0, ErrInvalidBulkStringLength
	}

	if n == -1 { // fast-path
		return rw.w.Write(nilBulkStringHeaderBytes)
	}

	return rw.writeNumber('$', int64(n))
}

// WriteBulkString writes the string s as bulk string.
//
// If you need to write a nil bulk string, use WriteBulkStringBytes instead.
func (rw *Writer) WriteBulkString(s string) (int, error) {
	n, err := rw.WriteBulkStringHeader(len(s))
	if err != nil {
		return n, err
	}

	rw.buf = rw.buf[:0]
	rw.buf = append(rw.buf, s...)
	rw.buf = append(rw.buf, '\r', '\n')

	n1, err := rw.w.Write(rw.buf)
	return n + n1, err
}

// WriteBulkStringBytes writes the byte slice s as bulk string.
func (rw *Writer) WriteBulkStringBytes(s []byte) (int, error) {
	if s == nil {
		return rw.WriteBulkStringHeader(-1)
	}

	n, err := rw.WriteBulkStringHeader(len(s))
	if err != nil {
		return n, err
	}

	rw.buf = rw.buf[:0]
	rw.buf = append(rw.buf, s...)
	rw.buf = append(rw.buf, '\r', '\n')

	n1, err := rw.w.Write(rw.buf)
	return n + n1, err
}

// WriteError writes the string s unvalidated as a simple error.
func (rw *Writer) WriteError(s string) (int, error) {
	return rw.writeString('-', s)
}

// WriteErrorBytes writes the byte slice s unvalidated as a simple error.
func (rw *Writer) WriteErrorBytes(s []byte) (int, error) {
	return rw.writeBytes('-', s)
}

// WriteInteger writes the integer i as the native RESP integer type.
func (rw *Writer) WriteInteger(i int) (int, error) {
	return rw.writeNumber(':', int64(i))
}

// WriteSimpleString writes the string s unvalidated as a simple string.
func (rw *Writer) WriteSimpleString(s string) (int, error) {
	return rw.writeString('+', s)
}

// WriteSimpleStringBytes writes the byte slice s unvalidated as a simple string.
func (rw *Writer) WriteSimpleStringBytes(s []byte) (int, error) {
	return rw.writeBytes('+', s)
}
