package engine

// PacketType indicates type of an engine.io Packet
type PacketType byte

const (
	// PacketTypeOpen is sent from the server when a new transport is opened (recheck)
	PacketTypeOpen PacketType = iota
	// PacketTypeClose requests the close of this transport but does not shutdown the connection itself.
	PacketTypeClose
	// PacketTypePing is sent by the client. Server should answer with a pong packet containing the same data
	PacketTypePing
	// PacketTypePong is sent by the server to respond to ping packets.
	PacketTypePong
	// PacketTypeMessage denotes actual message, client and server should call their callbacks with the data.
	PacketTypeMessage
	// PacketTypeUpgrade is sent by the client requesting the server to flush its cache on the old transport and switch to the new transport.
	PacketTypeUpgrade
	// PacketTypeNoop denotes a noop packet. Used primarily to force a poll cycle when an incoming websocket connection is received.
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
	// MessageTypeString indicates Message encoded as string
	MessageTypeString MessageType = iota
	// MessageTypeBinary indicates Message encoded as binary
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
