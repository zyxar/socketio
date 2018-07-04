package engio

import (
	"bufio"
	"encoding/base64"
	"errors"
	"io"
	"strconv"
)

var (
	errInvalidPayload = errors.New("invalid payload")
)

type PayloadReader interface {
	io.Reader
	io.ByteReader
}

type Payload struct {
	packets []Packet
}

func (p *Payload) ReadFrom(r io.Reader) (n int64, err error) {
	if rd, ok := r.(PayloadReader); ok {
		return p.readFrom(rd)
	}
	return p.readFrom(bufio.NewReader(r))
}

func (p *Payload) readFrom(r PayloadReader) (n int64, err error) {
	for {
		var pkt Packet
		nn, err := pkt.Decode(r)
		n += int64(nn)
		if err != nil {
			if err == io.EOF {
				return n, nil
			}
			return n, err
		}
		p.packets = append(p.packets, pkt)
	}
	return
}

func (p Payload) WriteTo(w io.Writer) (n int64, err error) {
	if len(p.packets) == 0 {
		return
	}
	var nn int64
	for i := range p.packets {
		nn, err = p.packets[i].WriteTo(w)
		n += nn
		if err != nil {
			return
		}
	}
	return
}

type Packet struct {
	msgType MessageType
	pktType PacketType
	data    []byte
}

func (p *Packet) encode() (int, []byte, []byte) {
	length := len(p.data)
	var div []byte
	dst := p.data
	switch p.msgType {
	case MessageTypeString:
		div = []byte{':', byte(p.pktType) + '0'}
		length += 1
	case MessageTypeBinary:
		length = base64.StdEncoding.EncodedLen(length)
		div = []byte{':', 'b', byte(p.pktType) + '0'}
		dst = make([]byte, length)
		base64.StdEncoding.Encode(dst, p.data)
		length += 2
	default:
		panic("invalid message type")
	}
	return length, div, dst
}

func (p *Packet) Encode() (encoded []byte) {
	length, div, data := p.encode()
	ls := strconv.FormatInt(int64(length), 10)
	ll := len(ls)
	payloadLength := length + ll
	encoded = make([]byte, ll, payloadLength)
	copy(encoded, ls)
	encoded = append(encoded, div...)
	encoded = append(encoded, data...)
	return
}

func (p *Packet) WriteTo(w io.Writer) (n int64, err error) {
	var nn int
	length, div, data := p.encode()
	ls := strconv.FormatInt(int64(length), 10)
	nn, err = io.WriteString(w, ls)
	n += int64(nn)
	if err != nil {
		return
	}
	nn, err = w.Write(div)
	n += int64(nn)
	if err != nil {
		return
	}
	nn, err = w.Write(data)
	n += int64(nn)
	return
}

func (p *Packet) Decode(pr PayloadReader) (int, error) {
	n, l, err := p.decode(pr)
	if err != nil {
		return n, err
	}
	var rd io.Reader = pr
	n += l
	if p.msgType == MessageTypeBinary {
		rd = base64.NewDecoder(base64.StdEncoding, pr /*io.LimitReader(pr, int64(l))*/)
		l = base64.StdEncoding.DecodedLen(l)
	}
	p.data = make([]byte, l)
	if l == 0 {
		return n, nil
	}
	_, err = rd.Read(p.data)
	return n, err
}

func (p *Packet) decode(r io.ByteReader) (n int, length int, err error) {
	for {
		b, err := r.ReadByte()
		if err != nil {
			return n, 0, err
		}
		n++
		if b == ':' {
			break
		}
		if b < '0' || b > '9' {
			return n, 0, errInvalidPayload
		}
		length = length*10 + int(b-'0')
	}
	b, err := r.ReadByte()
	if err != nil {
		return
	}
	n++
	if b == 'b' {
		length--
		b, err = r.ReadByte()
		if err != nil {
			return
		}
		n++
		p.msgType = MessageTypeBinary
	} else {
		p.msgType = MessageTypeString
	}
	length--
	p.pktType = PacketType(b - '0')
	return
}
