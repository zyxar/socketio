package engine

// PacketType indicates type of an engine.io Packet
type PacketType byte

const (
	PacketTypeOpen PacketType = iota
	PacketTypeClose
	PacketTypePing
	PacketTypePong
	PacketTypeMessage
	PacketTypeUpgrade
	PacketTypeNoop
)

// String returns string representation of a PacketType
func (p PacketType) String() string {
	switch p {
	case PacketTypeOpen:
		return "open"
	case PacketTypeClose:
		return "close"
	case PacketTypePing:
		return "ping"
	case PacketTypePong:
		return "pong"
	case PacketTypeMessage:
		return "message"
	case PacketTypeUpgrade:
		return "upgrade"
	case PacketTypeNoop:
		return "noop"
	}
	return "invalid"
}

// Parameters describes engine.io connection attributes, sending from server to client upon handshaking.
type Parameters struct {
	SID          string   `json:"sid"`
	Upgrades     []string `json:"upgrades"`
	PingInterval int      `json:"pingInterval"`
	PingTimeout  int      `json:"pingTimeout"`
}

// MessageType indicates type of an engine.io Message
type MessageType byte

const (
	MessageTypeString MessageType = iota
	MessageTypeBinary
)

// String returns string representation of a MessageType
func (m MessageType) String() string {
	switch m {
	case MessageTypeString:
		return "string"
	case MessageTypeBinary:
		return "binary"
	}
	return "invalid"
}

const (
	queryTransport = "transport"
	queryJSONP     = "j"
	querySession   = "sid"
	queryBase64    = "b64"
	queryEIO       = "EIO"

	// Version is engine.io-protocol version
	Version = "3"
)
