package socketio

const (
	Revision = "4"
)

type PacketType byte

const (
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
