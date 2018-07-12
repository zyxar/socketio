package socketio

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"

	"github.com/zyxar/socketio/engine"
)

var (
	ErrUnknownPacket = errors.New("unknown packet")
)

type Packet struct {
	Type      PacketType
	Namespace string
	Data      interface{}
	ID        *uint64

	event       *eventArgs
	attachments int
}

type eventArgs struct {
	name string
	data []byte
}

type üWriter interface {
	io.Writer
	io.ByteWriter
}

type Encoder interface {
	Encode(p *Packet) ([]byte, error)
}

type Decoder interface {
	Add(msgType MessageType, data []byte) error
	Decoded() <-chan *Packet
}

type Parser interface {
	Encoder() Encoder
	Decoder() Decoder
}

type defaultParser struct{}

var DefaultParser Parser = &defaultParser{}

func (defaultParser) Encoder() Encoder {
	return &defaultEncoder{}
}

func (defaultParser) Decoder() Decoder {
	return newDefaultDecoder()
}

type defaultEncoder struct{}

func (d defaultEncoder) Encode(p *Packet) ([]byte, error) {
	var buf bytes.Buffer
	if err := d.encodeTo(&buf, p); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (defaultEncoder) encodeTo(w üWriter, p *Packet) (err error) {
	if err = w.WriteByte(byte(p.Type) + '0'); err != nil {
		return
	}
	if p.Namespace != "" && p.Namespace != "/" {
		if _, err = io.WriteString(w, p.Namespace); err != nil {
			return
		}
		if err = w.WriteByte(','); err != nil {
			return
		}
	}
	if p.ID != nil {
		if _, err = io.WriteString(w, strconv.FormatUint(*p.ID, 10)); err != nil {
			return
		}
	}
	if p.Data != nil {
		err = json.NewEncoder(w).Encode(p.Data)
	}
	return
}

type defaultDecoder struct {
	packets chan *Packet
	lastp   *Packet
}

func newDefaultDecoder() *defaultDecoder {
	return &defaultDecoder{
		packets: make(chan *Packet, 8),
	}
}

func (d *defaultDecoder) Decoded() <-chan *Packet {
	return d.packets
}

func (d *defaultDecoder) Add(msgType MessageType, data []byte) error {
	if msgType != MessageTypeString {
		if d.lastp == nil {
			return ErrUnknownPacket
		}
		d.lastp.attachments-- // ignore binary data
	} else {
		p, err := d.decode(data)
		if err != nil {
			return err
		}
		d.lastp = p
	}

	if d.lastp.attachments == 0 {
		d.emit()
	}

	return nil
}

func (d *defaultDecoder) emit() {
	select {
	case d.packets <- d.lastp:
		d.lastp = nil
	}
}

func (defaultDecoder) decode(s []byte) (p *Packet, err error) {
	b := PacketType(s[0] - '0')
	if b > PacketTypeBinaryAck {
		return nil, ErrUnknownPacket
	}
	p = &Packet{Type: b}
	i := 1 // skip 1st byte

	if p.Type == PacketTypeBinaryEvent || p.Type == PacketTypeBinaryAck {
		j := i
		for ; j < len(s); j++ {
			if s[j] == '-' {
				break
			}
			if s[j] < '0' || s[j] > '9' {
				return nil, ErrUnknownPacket
			}
			p.attachments = p.attachments*10 + int(s[j]-'0')
		}
		i = j + 1
		if i >= len(s) {
			return p, nil
		}
	}

	if s[i] == '/' { // decode nsp
		j := i + 1
		for ; j < len(s); j++ {
			if s[j] == ',' {
				break
			}
		}
		p.Namespace = string(s[i:j])
		i = j + 1
		if i >= len(s) {
			return p, nil
		}
	} else {
		p.Namespace = "/"
	}
	if s[i] >= '0' && s[i] <= '9' { // decode id
		j := i + 1
		var id uint64 = uint64(s[i] - '0')
		for ; j < len(s); j++ {
			if s[j] >= '0' && s[j] <= '9' {
				id = id*10 + uint64(s[j]-'0')
			} else {
				break
			}
		}
		p.ID = newid(id)
		i = j
		if i >= len(s) {
			return p, nil
		}
	}
	if p.Type == PacketTypeEvent || p.Type == PacketTypeBinaryEvent { // extracts event but leaves data
		if s[i] == '[' {
			var event string
			j := i + 1
			for j < len(s) && s[j] != ',' && s[j] != ']' {
				j++
			}
			if j > i+1 {
				if err = json.Unmarshal(s[i+1:j], &event); err != nil {
					return nil, err
				}
				if s[j] == ',' {
					j++
				}
				dst := make([]byte, len(s[j:])+1)
				dst[0] = '['
				copy(dst[1:], s[j:])
				p.event = &eventArgs{event, dst}
			}
		}
		return p, nil
	} else if p.Type == PacketTypeAck || p.Type == PacketTypeBinaryAck {
		p.event = &eventArgs{data: s[i:]}
		return p, nil
	}

	return p, json.Unmarshal(s[i:], &p.Data)
}

func newid(id uint64) *uint64 {
	i := new(uint64)
	*i = id
	return i
}

type MessageType = engine.MessageType

const MessageTypeString MessageType = engine.MessageTypeString
const MessageTypeBinary MessageType = engine.MessageTypeBinary
