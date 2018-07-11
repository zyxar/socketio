package socketio

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
)

var (
	ErrUnknownPacket = errors.New("unknown packet")
)

type Packet struct {
	Type      PacketType
	Namespace string
	Data      interface{}
	ID        *uint64
	event     *eventArgs
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
	Decode(s []byte) (p *Packet, err error)
}

type Parser interface {
	Encoder
	Decoder
}

type defaultParser struct {
}

var DefaultParser Parser = &defaultParser{}

func (d defaultParser) Encode(p *Packet) ([]byte, error) {
	var buf bytes.Buffer
	if err := d.encodeTo(&buf, p); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (defaultParser) encodeTo(w üWriter, p *Packet) (err error) {
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

func (defaultParser) Decode(s []byte) (p *Packet, err error) {
	b := PacketType(s[0] - '0')
	if b > PacketTypeBinaryAck {
		return nil, ErrUnknownPacket
	}
	p = &Packet{Type: b}
	i := 1           // skip 1st byte
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
	if p.Type == PacketTypeEvent { // extracts event but leaves data
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
	} else if p.Type == PacketTypeAck {
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
