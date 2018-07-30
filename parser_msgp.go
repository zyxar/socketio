package socketio

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/tinylib/msgp/msgp"
)

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

func msgpUnmashalArg(i reflect.Value, data []byte) ([]byte, error) {
	var err error
	switch t := i.Interface().(type) {
	case msgp.Unmarshaler:
		data, err = t.UnmarshalMsg(data)
		return data, err
	case msgp.Extension:
		data, err = msgp.ReadExtensionBytes(data, t)
		return data, err
	case *bool:
		*t, data, err = msgp.ReadBoolBytes(data)
		return data, err
	case *float32:
		*t, data, err = msgp.ReadFloat32Bytes(data)
		return data, err
	case *float64:
		*t, data, err = msgp.ReadFloat64Bytes(data)
		return data, err
	case *complex64:
		*t, data, err = msgp.ReadComplex64Bytes(data)
		return data, err
	case *complex128:
		*t, data, err = msgp.ReadComplex128Bytes(data)
		return data, err
	case *uint8:
		*t, data, err = msgp.ReadUint8Bytes(data)
		return data, err
	case *uint16:
		*t, data, err = msgp.ReadUint16Bytes(data)
		return data, err
	case *uint32:
		*t, data, err = msgp.ReadUint32Bytes(data)
		return data, err
	case *uint64:
		*t, data, err = msgp.ReadUint64Bytes(data)
		return data, err
	case *uint:
		*t, data, err = msgp.ReadUintBytes(data)
		return data, err
	case *int8:
		*t, data, err = msgp.ReadInt8Bytes(data)
		return data, err
	case *int16:
		*t, data, err = msgp.ReadInt16Bytes(data)
		return data, err
	case *int32:
		*t, data, err = msgp.ReadInt32Bytes(data)
		return data, err
	case *int64:
		*t, data, err = msgp.ReadInt64Bytes(data)
		return data, err
	case *int:
		*t, data, err = msgp.ReadIntBytes(data)
		return data, err
	case *time.Duration:
		var vv int64
		vv, data, err = msgp.ReadInt64Bytes(data)
		return data, err
		*t = time.Duration(vv)
	case *time.Time:
		*t, data, err = msgp.ReadTimeBytes(data)
		return data, err
	case *string:
		*t, data, err = msgp.ReadStringBytes(data)
		return data, err
	case *[]byte:
		*t, data, err = msgp.ReadBytesZC(data)
		return data, err
	case *map[string]interface{}:
		*t, data, err = msgp.ReadMapStrIntfBytes(data, nil)
		return data, err
	case *interface{}:
		*t, data, err = msgp.ReadIntfBytes(data)
		return data, err
	default:
	}

	ie := i.Elem()
	switch ie.Kind() {
	case reflect.Ptr:
		vv := reflect.New(ie.Type().Elem())
		data, err = msgpUnmashalArg(vv, data)
		if err == nil {
			ie.Set(vv)
		}
		return data, err
	case reflect.Slice:
		var sz uint32
		sz, data, err = msgp.ReadArrayHeaderBytes(data)
		if err != nil {
			return data, err
		}
		slice := reflect.MakeSlice(ie.Type(), 0, int(sz))
		for i := 0; i < int(sz); i++ {
			vv := reflect.New(ie.Type().Elem())
			data, err = msgpUnmashalArg(vv, data)
			if err != nil {
				return data, err
			}
			slice = reflect.Append(slice, vv.Elem())
		}
		ie.Set(slice)
		return data, nil
	case reflect.Array:
		var sz uint32
		sz, data, err = msgp.ReadArrayHeaderBytes(data)
		if err != nil {
			return data, err
		}
		if ie.Len() < int(sz) {
			return data, fmt.Errorf("%s len is %d, but unmarshaling %d elements", ie.Type(), ie.Len(), sz)
		}
		for i := 0; i < int(sz); i++ {
			data, err = msgpUnmashalArg(ie.Index(i).Addr(), data)
			if err != nil {
				return data, err
			}
		}
		return data, nil
	case reflect.Interface:
		return data, fmt.Errorf("%v: not concrete type", ie.Type())
	case reflect.Invalid, reflect.Chan, reflect.Func, reflect.UnsafePointer, reflect.Struct:
		return data, fmt.Errorf("%v unsuppported", ie.Type())
	case reflect.Map:
		mapType := ie.Type()
		if mapType.Key().Kind() != reflect.String {
			return data, fmt.Errorf("%v unsuppported", mapType)
		}
		switch mapType.Elem().Kind() {
		case reflect.Invalid, reflect.Chan, reflect.Interface, reflect.Func, reflect.UnsafePointer, reflect.Struct:
			return data, fmt.Errorf("%v unsuppported", mapType)
		}
		var vv reflect.Value
		vv, data, err = msgpReadMap(data, mapType)
		if err != nil {
			return data, err
		}
		ie.Set(vv)
		return data, nil
	}

	// vv, data, err := msgp.ReadIntfBytes(data)
	// if err != nil {
	// 	return data, err
	// }
	// v := reflect.ValueOf(vv)
	// if v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
	// 	v = v.Elem()
	// }
	// ie.Set(v)
	return data, fmt.Errorf("%v unsuppported", ie.Type())
}

func msgpReadMap(b []byte, mapType reflect.Type) (v reflect.Value, o []byte, err error) {
	var sz uint32
	o = b
	if sz, o, err = msgp.ReadMapHeaderBytes(o); err != nil {
		return
	}
	v = reflect.MakeMapWithSize(mapType, int(sz))
	valType := mapType.Elem()
	for z := uint32(0); z < sz; z++ {
		if len(o) < 1 {
			err = msgp.ErrShortBytes
			return
		}
		var key []byte
		key, o, err = msgp.ReadMapKeyZC(o)
		if err != nil {
			return
		}
		val := reflect.New(valType)
		o, err = msgpUnmashalArg(val, o)
		if err != nil {
			return
		}
		v.SetMapIndex(reflect.ValueOf(string(key)), val.Elem())
	}
	return
}

func (msgpackDecoder) UnmarshalArgs(args []reflect.Type, data []byte, _ [][]byte) (in []reflect.Value, err error) {
	var sz uint32
	sz, data, err = msgp.ReadArrayHeaderBytes(data)
	if err != nil {
		return
	}
	if len(args) > int(sz) {
		err = fmt.Errorf("not enough data to init %d arguments but only %d data", len(args), sz)
		return
	}

	in = make([]reflect.Value, len(args))
	for i, typ := range args {
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		in[i] = reflect.New(typ)
		data, err = msgpUnmashalArg(in[i], data)
		if err != nil {
			return
		}
		if args[i].Kind() != reflect.Ptr {
			in[i] = in[i].Elem()
		}
	}

	return in, nil
}

func (msgpackDecoder) ParseData(p *Packet) (event string, data []byte, bin [][]byte, err error) {
	switch p.Type {
	case PacketTypeConnect, PacketTypeDisconnect, PacketTypeError:
		return
	default:
	}

	b, ok := p.Data.([]byte)
	if !ok {
		err = fmt.Errorf("data should be raw bytes but got %T", p.Data)
		return
	}
	if t := msgp.NextType(b); t != msgp.ArrayType {
		err = fmt.Errorf("data should be a list of arguments but got %v", t)
		return
	}
	data = b
	var sz uint32
	sz, b, err = msgp.ReadArrayHeaderBytes(b)
	if err != nil {
		return
	}
	switch p.Type {
	case PacketTypeEvent, PacketTypeBinaryEvent:
		{
			if t := msgp.NextType(b); t != msgp.StrType {
				err = fmt.Errorf("event name should have string type but got %v", t)
				return
			}
			event, b, err = msgp.ReadStringBytes(b)
			if err != nil {
				return
			}
			// reconstruct array
			data = make([]byte, 0, len(b))
			data = msgp.AppendArrayHeader(data, sz-1)
			data = append(data, b...)
		}
	case PacketTypeAck, PacketTypeBinaryAck:
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
