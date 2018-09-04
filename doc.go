// Package resp implements fast Reader and Writer txpes for parsing the Redis RESP protocol.
//
// This package is low level and only deals with parsing of the different messages in the RESP protocol, avoiding any
// kind of validation that would slow down reading / writing.
//
// All structs can be reused via the corresponding Reset method and duplex connections are supported using a ReadWriter
// type that wraps a Reader and a Writer in a single allocation.
package resp
