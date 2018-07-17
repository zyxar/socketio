package socketio

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
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
	buffer      [][]byte
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
	Encode(p *Packet) ([]byte, [][]byte, error)
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

func (p *Packet) preprocess() {
	if p.Namespace != "" && p.Namespace[0] != '/' {
		p.Namespace = "/" + p.Namespace
	}
	p.attachments = 0
	if d, ok := p.Data.([]interface{}); ok {
		p.buffer = make([][]byte, 0, len(d))
		for i := range d {
			if b, ok := d[i].(Binary); ok {
				d[i] = &binaryWrap{b, p.attachments}
				p.attachments++
				p.buffer = append(p.buffer, b.Marshal())
			}
		}
	}
	if p.attachments > 0 {
		switch p.Type {
		case PacketTypeEvent:
			p.Type = PacketTypeBinaryEvent
		case PacketTypeAck:
			p.Type = PacketTypeBinaryAck
		}
	}
}

type defaultEncoder struct{}

func (d defaultEncoder) Encode(p *Packet) ([]byte, [][]byte, error) {
	var buf bytes.Buffer
	if err := d.encodeTo(&buf, p); err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), p.buffer, nil
}

func (defaultEncoder) encodeTo(w üWriter, p *Packet) (err error) {
	p.preprocess()

	if err = w.WriteByte(byte(p.Type) + '0'); err != nil {
		return
	}
	if p.attachments > 0 {
		if _, err = io.WriteString(w, strconv.Itoa(p.attachments)); err != nil {
			return
		}
		if err = w.WriteByte('-'); err != nil {
			return
		}
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
		i := len(d.lastp.buffer) - d.lastp.attachments
		d.lastp.buffer[i] = data
		d.lastp.attachments--
	} else {
		p, err := d.decode(data)
		if err != nil {
			return err
		}
		if len(p.buffer) != p.attachments {
			return ErrUnknownPacket
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
	if len(s) < 1 {
		return nil, ErrUnknownPacket
	}
	b := PacketType(s[0] - '0')
	if b > PacketTypeBinaryAck {
		return nil, ErrUnknownPacket
	}
	p = &Packet{Type: b}
	i := 1 // skip 1st byte

	if i >= len(s) {
		return p, nil
	}

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
			text := s[i:]
			if p.Type == PacketTypeBinaryEvent {
				p.buffer, text = extractAttachments(text)
			}
			event, left, match := extractEvent(text)
			if !match {
				return nil, ErrUnknownPacket
			}
			p.event = &eventArgs{name: event, data: left}
		}
		return p, nil
	} else if p.Type == PacketTypeAck || p.Type == PacketTypeBinaryAck {
		text := s[i:]
		if s[i] == '[' {
			if p.Type == PacketTypeBinaryAck {
				p.buffer, text = extractAttachments(text)
			}
		}
		p.event = &eventArgs{data: text}
		return p, nil
	}

	return p, json.Unmarshal(s[i:], &p.Data)
}

func newid(id uint64) *uint64 {
	i := new(uint64)
	*i = id
	return i
}

func extractEvent(b []byte) (event string, left []byte, match bool) {
	if sub := eventExp.FindSubmatch(b); sub != nil {
		event = string(sub[1])
		left = eventExp.ReplaceAll(b, []byte{'['})
		match = true
	}
	return
}

func extractAttachments(b []byte) (buffer [][]byte, left []byte) {
	if m := placeholderExp.FindAllSubmatch(b, -1); m != nil {
		buffer = make([][]byte, len(m))
	}
	left = placeholderExp.ReplaceAllLiteral(b, nil)
	return
}

type MessageType = engine.MessageType

const MessageTypeString MessageType = engine.MessageTypeString
const MessageTypeBinary MessageType = engine.MessageTypeBinary

var placeholderExp = regexp.MustCompile(`\s*,\s*\{\s*"_placeholder"\s*:\s*true\s*,\s*"num"\s*:\s*\d*?\s*\}\s*`)
var eventExp = regexp.MustCompile(`^\[\s*"(?P<event>[^"]+)"\s*,?`)

type Binary interface {
	Marshal() []byte
	Unmarshal([]byte)
}

type binaryWrap struct {
	Binary
	num int
}

func (b binaryWrap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if _, err := fmt.Fprintf(&buf, `{"_placeholder":true,"num":%d}`, b.num); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type Bytes struct {
	Data []byte
}

func (b *Bytes) Marshal() []byte {
	return b.Data[:]
}

func (b *Bytes) Unmarshal(p []byte) {
	b.Data = p
}
