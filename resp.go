package resp

import (
	"errors"
	"fmt"
	"io"
)

var (
	// ErrInvalidArrayLength is returned when reading or writing an array header with an invalid length.
	ErrInvalidArrayLength = errors.New("array length must be >= -1")

	// ErrInvalidBulkStringLength is returned when reading or writing a bulk string with an invalid length.
	ErrInvalidBulkStringLength = errors.New("bulk string length must be >= -1")

	// ErrInvalidInteger is returned when decoding an invalid integer.
	ErrInvalidInteger = errors.New("invalid integer")

	// ErrUnexpectedEOL is returned when reading a line that does not end in \r.\n
	ErrUnexpectedEOL = errors.New("missing or invalid EOL")

	// ErrUnexpectedType is returned by Reader when encountering an unknown type.
	ErrUnexpectedType = errors.New("encountered unexpected RESP type")
)

// Type is an enum of the known RESP types with the values of the constants being the single-byte prefix characters.
type Type byte

const (
	// TypeInvalid is returned by Reader when encountering unknown or invalid types.
	TypeInvalid Type = 0
	// TypeArray signifies a RESP array.
	TypeArray Type = '*'
	// TypeBulkString signifies a RESP bulk string.
	TypeBulkString Type = '$'
	// TypeError signifies an error string.
	TypeError Type = '-'
	// TypeInteger signifies a integer.
	TypeInteger Type = ':'
	// TypeSimpleString signifies a simple string.
	TypeSimpleString Type = '+'
)

var _ fmt.Stringer = TypeInvalid

var types = [255]Type{
	TypeArray:        TypeArray,
	TypeBulkString:   TypeBulkString,
	TypeError:        TypeError,
	TypeInteger:      TypeInteger,
	TypeSimpleString: TypeSimpleString,
}

// String implements the fmt.Stringer interface.
func (t Type) String() string {
	return string(t)
}

// ReaderWriter embeds a Reader and a Writer in a single allocation for an io.ReadWriter.
//
// A single Reader and a single Writer method can be called concurrently, given the Read and Write methods of the
// underlying io.ReadWriter are safe for concurrent use.
type ReaderWriter struct {
	Reader
	Writer
}

// NewReaderWriter returns a new ReaderWriter that uses the given io.ReadWriter.
func NewReaderWriter(rw io.ReadWriter) *ReaderWriter {
	var rrw ReaderWriter
	rrw.Reset(rw)
	return &rrw
}

// Reset resets the embedded Reader and Writer to use the given io.ReadWriter.
//
// Reset must not be called concurrently with any other method
func (rrw *ReaderWriter) Reset(rw io.ReadWriter) {
	rrw.Reader.Reset(rw)
	rrw.Writer.Reset(rw)
}
