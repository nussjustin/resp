package resp

import (
	"errors"
	"fmt"
	"io"
)

var (
	// ErrInvalidArrayLength is returned when reading or writing an array header with an invalid length.
	ErrInvalidArrayLength = errors.New("array length must be >= -1")

	// ErrInvalidBlobStringLength is returned when reading or writing a blob string with an invalid length.
	ErrInvalidBlobStringLength = errors.New("blob string length must be >= -1")

	// ErrInvalidNumber is returned when decoding an invalid number.
	ErrInvalidNumber = errors.New("invalid number")

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
	// TypeBlobString signifies a RESP blob string.
	TypeBlobString Type = '$'
	// TypeNumber signifies a number.
	TypeNumber Type = ':'
	// TypeError signifies an error string.
	TypeSimpleError Type = '-'
	// TypeSimpleString signifies a simple string.
	TypeSimpleString Type = '+'
)

var _ fmt.Stringer = TypeInvalid

var types = [255]Type{
	TypeArray:        TypeArray,
	TypeBlobString:   TypeBlobString,
	TypeNumber:      TypeNumber,
	TypeSimpleError:  TypeSimpleError,
	TypeSimpleString: TypeSimpleString,
}

// String implements the fmt.Stringer interface.
func (t Type) String() string {
	return string(t)
}

// ReadWriter embeds a Reader and a Writer in a single allocation for an io.ReadWriter.
//
// A single Reader and a single Writer method can be called concurrently, given the Read and Write methods of the
// underlying io.ReadWriter are safe for concurrent use.
type ReadWriter struct {
	Reader
	Writer
}

// NewReadWriter returns a new ReadWriter that uses the given io.ReadWriter.
func NewReadWriter(rw io.ReadWriter) *ReadWriter {
	var rrw ReadWriter
	rrw.Reset(rw)
	return &rrw
}

// Reset resets the embedded Reader and Writer to use the given io.ReadWriter.
//
// Reset must not be called concurrently with any other method
func (rrw *ReadWriter) Reset(rw io.ReadWriter) {
	rrw.Reader.Reset(rw)
	rrw.Writer.Reset(rw)
}
