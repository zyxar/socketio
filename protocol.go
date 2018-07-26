//go:generate msgp -tests=false

package socketio

const (
	// Revision is protocol version
	Revision = "4"
)

// PacketType indicates type of a Packet
type PacketType byte

const ( // PacketType enums
	PacketTypeConnect PacketType = iota
	PacketTypeDisconnect
	PacketTypeEvent
	PacketTypeAck
	PacketTypeError
	PacketTypeBinaryEvent
	PacketTypeBinaryAck
)

func (p PacketType) String() string {
	switch p {
	case PacketTypeConnect:
		return "CONNECT"
	case PacketTypeDisconnect:
		return "DISCONNECT"
	case PacketTypeEvent:
		return "EVENT"
	case PacketTypeAck:
		return "ACK"
	case PacketTypeError:
		return "ERROR"
	case PacketTypeBinaryEvent:
		return "BINARY_EVENT"
	case PacketTypeBinaryAck:
		return "BINARY_ACK"
	}
	return "INVALID"
}

// Packet is message abstraction, representing for data exchanged between socket.io server and client
type Packet struct {
	Type      PacketType  `msg:"type" json:"type"`
	Namespace string      `msg:"nsp" json:"nsp"`
	Data      interface{} `msg:"data" json:"data,omitempty"`
	ID        *uint64     `msg:"id" json:"id,omitempty"`

	event       *eventArgs
	attachments int
	buffer      [][]byte
}
