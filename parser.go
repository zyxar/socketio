package socketio

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/zyxar/socketio/engine"

	"github.com/tinylib/msgp/msgp"
)

var (
	// ErrUnknownPacket indicates packet invalid or unknown when parser encoding/decoding data
	ErrUnknownPacket = errors.New("unknown packet")
)

type eventArgs struct {
	name string
	data []byte
}

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
}

// Parser provides Encoder and Decoder instance, like a factory
type Parser interface {
	Encoder() Encoder
	Decoder() Decoder
}

type defaultParser struct{}

// DefaultParser is default parser implementation for socket.io, compatible with `socket.io-parser`.
var DefaultParser Parser = &defaultParser{}

func (defaultParser) Encoder() Encoder {
	return &defaultEncoder{}
}

func (defaultParser) Decoder() Decoder {
	return newDefaultDecoder()
}

type defaultEncoder struct{}

func (d defaultEncoder) Encode(p *Packet) ([]byte, [][]byte, error) {
	var buf bytes.Buffer
	if err := d.encodeTo(&buf, p); err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), p.buffer, nil
}

func (defaultEncoder) preprocess(p *Packet) {
	if p.Namespace != "" && p.Namespace[0] != '/' {
		p.Namespace = "/" + p.Namespace
	}
	p.attachments = 0
	if d, ok := p.Data.([]interface{}); ok {
		p.buffer = make([][]byte, 0, len(d))
		for i := range d {
			if b, ok := d[i].(encoding.BinaryMarshaler); ok {
				d[i] = &placeholder{num: p.attachments}
				p.attachments++
				bb, _ := b.MarshalBinary()
				p.buffer = append(p.buffer, bb)
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

func (d defaultEncoder) encodeTo(w üWriter, p *Packet) (err error) {
	d.preprocess(p)

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

func (defaultDecoder) ParseData(p *Packet) (string, []byte, [][]byte, error) {
	if p.event == nil {
		return "", nil, nil, nil
	}
	return p.event.name, p.event.data, p.buffer, nil
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
	p = &Packet{Type: b, Namespace: "/"}
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
	}

	if s[i] >= '0' && s[i] <= '9' { // decode id
		j := i + 1
		var id = uint64(s[i] - '0')
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

// MessageType is alias of engine.MessageType
type MessageType = engine.MessageType

const (
	// MessageTypeString is alias of engine.MessageTypeString
	MessageTypeString MessageType = engine.MessageTypeString
	// MessageTypeBinary is alias of engine.MessageTypeBinary
	MessageTypeBinary MessageType = engine.MessageTypeBinary
)

var placeholderExp = regexp.MustCompile(`\s*,\s*\{\s*"_placeholder"\s*:\s*true\s*,\s*"num"\s*:\s*\d*?\s*\}\s*`)
var eventExp = regexp.MustCompile(`^\[\s*"(?P<event>[^"]+)"\s*,?`)

// Binary refers to binary data to be exchanged between socket.io server and client
type Binary interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type placeholder struct {
	num int
}

func (b placeholder) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if _, err := fmt.Fprintf(&buf, `{"_placeholder":true,"num":%d}`, b.num); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Bytes is default implementation of Binary interface, a helper to transfer `[]byte`
type Bytes struct {
	Data []byte
}

// Marshal implements Binary interface
func (b Bytes) MarshalBinary() ([]byte, error) {
	return b.Data[:], nil
}

// Unmarshal implements Binary interface
func (b *Bytes) UnmarshalBinary(p []byte) error {
	b.Data = p
	return nil
}

// MarshalBinaryTo copies data into 'p', implementing msgp.Extension.MarshalBinaryTo
func (b *Bytes) MarshalBinaryTo(p []byte) error {
	copy(p, b.Data)
	return nil
}

// parser - msgp

var MsgpackParser Parser = &msgpackParser{}

type msgpackParser struct{}
type msgpackEncoder struct{}
type msgpackDecoder struct{ packets chan *Packet }

func (msgpackParser) Encoder() Encoder { return &msgpackEncoder{} }
func (msgpackParser) Decoder() Decoder { return newMsgpackDecoder(8) }

func (msgpackEncoder) Encode(p *Packet) ([]byte, [][]byte, error) {
	switch p.Type {
	case PacketTypeConnect, PacketTypeDisconnect, PacketTypeError:
		b, err := json.Marshal(p)
		return b, nil, err
	default:
	}
	o, err := p.MarshalMsg(nil)
	return nil, [][]byte{o}, err
}

func newMsgpackDecoder(size int) *msgpackDecoder {
	return &msgpackDecoder{packets: make(chan *Packet, size)}
}

func (msgpackDecoder) ParseData(p *Packet) (event string, data []byte, bin [][]byte, err error) {
	switch p.Type {
	case PacketTypeEvent, PacketTypeAck, PacketTypeBinaryEvent, PacketTypeBinaryAck:
		if d, ok := p.Data.([]interface{}); ok {
			var args []interface{}
			for i := range d {
				if raw, ok := d[i].(*msgp.RawExtension); ok {
					bin = append(bin, raw.Data)
				} else {
					args = append(args, d[i])
				}
			}
			if len(args) > 0 {
				if p.Type == PacketTypeEvent || p.Type == PacketTypeBinaryEvent {
					if evt, ok := args[0].(string); ok {
						event = evt
					} else {
						err = fmt.Errorf("first argument should be event name but got %T", args[0])
						return
					}
					args = args[1:]
				}
				data, err = json.Marshal(args)
				return
			}
		} else {
			err = fmt.Errorf("packet data should be an array but got %T", p.Data)
			return
		}
	default:
	}
	return
}

func (m *msgpackDecoder) Add(msgType MessageType, data []byte) (err error) {
	var p Packet
	switch msgType {
	case MessageTypeString:
		err = json.Unmarshal(data, &p)
	case MessageTypeBinary:
		_, err = p.UnmarshalMsg(data)
	}
	if err != nil {
		return err
	}
	if p.Namespace == "" {
		p.Namespace = "/"
	}
	m.packets <- &p
	return nil
}

func (m *msgpackDecoder) Decoded() <-chan *Packet { return m.packets }
