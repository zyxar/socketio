package engineio

type PacketType byte

const (
	PacketTypeOpen PacketType = iota
	PacketTypeClose
	PacketTypePing
	PacketTypePong
	PacketTypeMessage
	PacketTypeUpgrade
	PacketTypeNoop
	_PacketTypeMax
)

type Packet struct {
	PacketType
}

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

type Parameters struct {
	SID          string   `json:"sid"`
	Upgrades     []string `json:"upgrades"`
	PingInterval int      `json:"pingInterval"`
	PingTimeout  int      `json:"pingTimeout"`
}
