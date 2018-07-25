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
		return "Packet#CONNECT (0)"
	case PacketTypeDisconnect:
		return "Packet#DISCONNECT (1)"
	case PacketTypeEvent:
		return "Packet#EVENT (2)"
	case PacketTypeAck:
		return "Packet#ACK (3)"
	case PacketTypeError:
		return "Packet#ERROR (4)"
	case PacketTypeBinaryEvent:
		return "Packet#BINARY_EVENT (5)"
	case PacketTypeBinaryAck:
		return "Packet#BINARY_ACK (6)"
	}
	return "Packet#INVALID"
}

// Packet is message abstraction, representing for data exchanged between socket.io server and client
type Packet struct {
	Type      PacketType  `msg:"type" json:"type"`
	Namespace string      `msg:"nsp" json:"nsp"`
	Data      interface{} `msg:"data" json:"data,omitempty"`
	ID        *uint64     `msg:"id" json:"id,omitempty"`

	event       *eventArgs `msg:"-" json:"-"`
	attachments int        `msg:"-" json:"-"`
	buffer      [][]byte   `msg:"-" json:"-"`
}
