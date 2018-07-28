package socketio

import (
	"errors"
	"io"
	"reflect"

	"github.com/zyxar/socketio/engine"
)

var (
	// ErrUnknownPacket indicates packet invalid or unknown when parser encoding/decoding data
	ErrUnknownPacket = errors.New("unknown packet")
	// DefaultParser is default parser implementation for socket.io, compatible with `socket.io-parser`.
	DefaultParser Parser = &defaultParser{}
	// MsgpackParser is msgpack parser implementation for socket.io, compatible with `socket.io-msgpack-parser`.
	MsgpackParser Parser = &msgpackParser{}
)

type üWriter interface {
	io.Writer
	io.ByteWriter
}

// Encoder encodes a Packet into byte format
type Encoder interface {
	Encode(p *Packet) ([]byte, [][]byte, error)
}

// Decoder decodes data into a Packet
type Decoder interface {
	Add(msgType MessageType, data []byte) error
	Decoded() <-chan *Packet
	ParseData(p *Packet) (string, []byte, [][]byte, error)
	ArgsUnmarshaler
}

type ArgsUnmarshaler interface {
	UnmarshalArgs([]reflect.Type, []byte, [][]byte) ([]reflect.Value, error)
}

// Parser provides Encoder and Decoder instance, like a factory
type Parser interface {
	Encoder() Encoder
	Decoder() Decoder
}

// MessageType is alias of engine.MessageType
type MessageType = engine.MessageType

const (
	// MessageTypeString is alias of engine.MessageTypeString
	MessageTypeString MessageType = engine.MessageTypeString
	// MessageTypeBinary is alias of engine.MessageTypeBinary
	MessageTypeBinary MessageType = engine.MessageTypeBinary
)
