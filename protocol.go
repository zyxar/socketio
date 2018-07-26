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
