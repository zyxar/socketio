package socketio

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strconv"
)

type defaultParser struct{}

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

func (defaultDecoder) ParseData(p *Packet) (event string, data []byte, bin [][]byte, err error) {
	text, ok := p.Data.([]byte)
	if !ok {
		err = fmt.Errorf("data should be bytes but got %T", p.Data)
		return
	}
	switch p.Type {
	case PacketTypeBinaryEvent:
		bin = p.buffer
		fallthrough
	case PacketTypeEvent:
		var match bool
		if event, data, match = extractEvent(text); !match {
			err = ErrUnknownPacket
		}
	case PacketTypeBinaryAck:
		bin = p.buffer
		data = text
	case PacketTypeAck:
		data = text
	}
	return
}

func (defaultDecoder) UnmarshalArgs(args []reflect.Type, data []byte, buffer [][]byte) ([]reflect.Value, error) {
	argv := make([]interface{}, 0, len(args))
	in := make([]reflect.Value, len(args))
	for i, typ := range args {
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		in[i] = reflect.New(typ)
		it := in[i].Interface()
		if b, ok := it.(encoding.BinaryUnmarshaler); ok {
			if len(buffer) > 0 {
				if err := b.UnmarshalBinary(buffer[0]); err != nil {
					return nil, err
				}
				buffer = buffer[1:]
			}
		} else {
			argv = append(argv, it)
		}
	}
	if err := json.Unmarshal(data, &argv); err != nil {
		return nil, err
	}
	for i := range args {
		if args[i].Kind() != reflect.Ptr {
			in[i] = in[i].Elem()
		}
	}
	return in, nil
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

	switch p.Type {
	case PacketTypeEvent, PacketTypeAck:
		if s[i] != '[' {
			err = fmt.Errorf("data should be a list of arguments but got %c", s[i])
			return
		}
		p.Data = s[i:]
	case PacketTypeBinaryAck, PacketTypeBinaryEvent:
		if s[i] != '[' {
			err = fmt.Errorf("data should be a list of arguments but got %c", s[i])
			return
		}
		p.Data = s[i:]
		p.buffer, p.Data = extractAttachments(s[i:])
	default:
		err = json.Unmarshal(s[i:], &p.Data)
	}

	return
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

var placeholderExp = regexp.MustCompile(`\s*,\s*\{\s*"_placeholder"\s*:\s*true\s*,\s*"num"\s*:\s*\d*?\s*\}\s*`)
var eventExp = regexp.MustCompile(`^\[\s*"(?P<event>[^"]+)"\s*,?`)

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

// Binary refers to binary data to be exchanged between socket.io server and client
type Binary interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
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
