package socketio

import (
	"bufio"
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
}

type üWriter interface {
	io.Writer
	io.ByteWriter
}

type Encoder interface {
	EncodeTo(w üWriter, p *Packet) error
	Encode(p *Packet) ([]byte, error)
}

type Decoder interface {
	Decode(s []byte) (p *Packet, err error)
	DecodeFrom(rd io.Reader) (p *Packet, err error)
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
	if err := d.EncodeTo(&buf, p); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (defaultParser) EncodeTo(w üWriter, p *Packet) (err error) {
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
	return p, json.Unmarshal([]byte(s[i:]), &p.Data)
}

func (defaultParser) DecodeFrom(rd io.Reader) (p *Packet, err error) {
	r := bufio.NewReader(rd)
	b, err := r.ReadByte()
	if err != nil {
		return
	}
	t := PacketType(b - '0')
	if t > PacketTypeBinaryAck {
		err = ErrUnknownPacket
		return
	}
	p = &Packet{Type: t}
	b, err = r.ReadByte()
	if err != nil {
		return
	}
	if b == '/' { // decode nsp
		nsp, err := r.ReadString(',')
		if err != nil {
			if err == io.EOF {
				p.Namespace = "/" + nsp
				return p, nil
			}
			return nil, err
		}
		p.Namespace = "/" + nsp[:len(nsp)-1]
	} else {
		r.UnreadByte()
		p.Namespace = "/"
	}

	var id uint64
	for { // decode id
		b, err = r.ReadByte()
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return
		}
		if b < '0' || b > '9' {
			r.UnreadByte()
			break
		}
		id = id*10 + uint64(b-'0')
	}
	p.ID = newid(id)
	err = json.NewDecoder(r).Decode(&p.Data)
	if err == io.EOF {
		err = nil
	}
	return
}

func newid(id uint64) *uint64 {
	i := new(uint64)
	*i = id
	return i
}
