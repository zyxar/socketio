package socketio

import (
	"bytes"
	"testing"
	"unsafe"

	"github.com/tinylib/msgp/msgp"
)

func TestParserEncodeDecodeString(t *testing.T) {
	var packets = []Packet{
		{Type: PacketTypeConnect, Namespace: "/woot"},
		{Type: PacketTypeDisconnect, Namespace: "/woot"},
		{Type: PacketTypeEvent, Namespace: "/", Data: []interface{}{"abcdefg", 1, empty}},
		{Type: PacketTypeEvent, Namespace: "/test", Data: []interface{}{"abcdefg", 1, empty}, ID: newid(1)},
		{Type: PacketTypeEvent, Namespace: "/test", Data: []interface{}{"abcdefg"}, ID: newid(2)},
		{Type: PacketTypeAck, Namespace: "/", Data: []interface{}{"a", 1, empty}, ID: newid(123)},
		{Type: PacketTypeError, Namespace: "/", Data: "Unauthorized"},
	}
	var encodedData = []string{"0/woot,", "1/woot,", `2["abcdefg",1,{}]
`, `2/test,1["abcdefg",1,{}]
`, `2/test,2["abcdefg"]
`, `3123["a",1,{}]
`, `4"Unauthorized"
`}

	encoder := DefaultParser.Encoder()
	decoder := DefaultParser.Decoder()

	for i := range packets {
		if encoded, _, err := encoder.Encode(&packets[i]); err != nil {
			t.Error(i, err.Error())
		} else if string(encoded) != encodedData[i] {
			t.Errorf("%d: %q != %q", i, encoded, encodedData[i])
		}
	}

	for i := range encodedData {
		err := decoder.Add(MessageTypeString, []byte(encodedData[i]))
		if err != nil {
			t.Error(i, err.Error())
		}
		p := <-decoder.Decoded()
		if !equalPacket(p, &packets[i]) {
			t.Errorf("%d: %s decoded error", i, encodedData[i])
		}
	}
}

func equalPacket(p1, p2 *Packet) bool {
	if p1 == p2 {
		return true
	}
	if p1 == nil || p2 == nil {
		return false
	}
	if p1.Type != p2.Type || p1.Namespace != p2.Namespace {
		return false
	}
	var id1, id2 uint64
	if p1.ID != nil {
		id1 = *p1.ID
	}
	if p2.ID != nil {
		id2 = *p2.ID
	}
	if id1 != id2 {
		return false
	}
	return true
}

var empty = map[string]interface{}{}

func TestBinaryEventDecode(t *testing.T) {
	text := []byte(`[
  "abcdefg",
  {
    "_placeholder": true,
    "num": 0
  },
  {
    "_placeholder": true,
    "num": 1
  },
  {
    "_placeholder": true,
    "num": 2
  },
  {
    "_placeholder": true,
    "num": 3
  },
  {
    "_placeholder": true,
    "num": 4
  },
  {
    "_placeholder": true,
    "num": 12
  },
  {
    "_placeholder": true,
    "num": "A"
  },
  1,
  2
]`)

	buffer, left := extractAttachments(text)
	if len(buffer) != 6 {
		t.Error("extract attachments incorrect", len(buffer))
	}

	event, _, match := extractEvent(left)
	if !match || event != "abcdefg" {
		t.Error("extract event incorrect")
	}
}

func TestParserEncodeBinary(t *testing.T) {
	encoder := DefaultParser.Encoder()
	b := [][]byte{{1, 2, 3, 4}, {2, 3, 4, 6}, {4, 5, 6, 8}}
	p := &Packet{Type: PacketTypeBinaryEvent, Namespace: "/", Data: []interface{}{"message",
		&Bytes{Data: b[0]},
		&Bytes{Data: b[1]},
		"TEXT",
		&Bytes{Data: b[2]},
	}, ID: newid(1)}
	encodedString := `53-1["message",{"_placeholder":true,"num":0},{"_placeholder":true,"num":1},"TEXT",{"_placeholder":true,"num":2}]
`
	encoded, bin, err := encoder.Encode(p)
	if err != nil {
		t.Error(err.Error())
	}
	if len(bin) != 3 {
		t.Error("encoded length incorrect")
	}
	if string(encoded) != encodedString {
		t.Error("encoded string packet incorrect")
	}
	for i, e := range bin {
		if !bytes.Equal(e, b[i]) {
			t.Error("encoded binary incorrect")
		}
	}
}

func TestMsgpackParseData(t *testing.T) {
	p := &Packet{Type: PacketTypeEvent, Data: []interface{}{"event", 1, "data"}}
	b, err := p.MarshalMsg(nil)
	if err != nil {
		t.Error(err.Error())
		return
	}
	_, err = p.UnmarshalMsg(b)
	if err != nil {
		t.Error(err.Error())
		return
	}
	event, data, _, err := (msgpackDecoder{}).ParseData(p)
	if err != nil {
		t.Error(err.Error())
	}
	if event != "event" {
		t.Error("event name incorrect")
	}
	_, o, err := msgp.ReadIntfBytes(data)
	if err != nil {
		t.Error(err.Error())
		return
	}
	if len(o) != 0 {
		t.Errorf("reconstruction data incorrect: remains %d bytes", len(o))
	}
}

func TestMsgpackUnmarshal(t *testing.T) {
	ints := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	strings := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"}
	bytess := [][]byte{{1}, {2}, {3}, {4}, {5}, {6}, {7}, {8}, {9}, {0}}
	foos := []*foobar{{"a"}, {"b"}, {"c"}, {"d"}}
	ptr := newid(12345)

	{
		data, err := msgp.AppendIntf(nil, []interface{}{ptr})
		if err != nil {
			t.Fatal(err.Error())
		}
		cb := newCallback(func(i **uint64) {
			if i == nil {
				t.Error("unmarshal *pointer incorrect")
			} else if *i == nil {
				t.Error("unmarshal *pointer incorrect")
			} else if **i != *ptr {
				t.Error("unmarshal *pointer incorrect")
			}
		})
		in, err := (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil)
		if err != nil {
			t.Error(err.Error())
			return
		}
		cb.fn.Call(in)
	}
	{
		data, err := msgp.AppendIntf(nil, []interface{}{ints})
		if err != nil {
			t.Fatal(err.Error())
		}
		cb := newCallback(func(i []int) {
			if len(i) != len(ints) || cap(i) != cap(ints) {
				t.Error("unmarshal []int incorrect")
			}
			for j := range i {
				if i[j] != ints[j] {
					t.Error("unmarshal []int", j, "incorrect")
				}
			}
		})
		in, err := (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil)
		if err != nil {
			t.Error(err.Error())
			return
		}
		cb.fn.Call(in)

		cb = newCallback(func(i [10]int) {
			if len(i) != len(ints) || cap(i) != cap(ints) {
				t.Error("unmarshal []int incorrect")
			}
			for j := range i {
				if i[j] != ints[j] {
					t.Error("unmarshal []int", j, "incorrect")
				}
			}
		})
		in, err = (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil)
		if err != nil {
			t.Error(err.Error())
			return
		}
		cb.fn.Call(in)

		cb = newCallback(func(i ...int) {
			if len(i) != len(ints) || cap(i) != cap(ints) {
				t.Error("unmarshal ...int incorrect")
			}
			for j := range i {
				if i[j] != ints[j] {
					t.Error("unmarshal ...int", j, "incorrect")
				}
			}
		})
		in, err = (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil)
		if err != nil {
			t.Error(err.Error())
			return
		}
		cb.fn.CallSlice(in)
	}
	{
		data, err := msgp.AppendIntf(nil, []interface{}{strings})
		if err != nil {
			t.Fatal(err.Error())
		}
		cb := newCallback(func(i []string) {
			if len(i) != len(strings) || cap(i) != cap(strings) {
				t.Error("unmarshal [10]string incorrect")
			}
			for j := range i {
				if i[j] != strings[j] {
					t.Error("unmarshal [10]string", j, "incorrect")
				}
			}
		})
		in, err := (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil)
		if err != nil {
			t.Error(err.Error())
			return
		}
		cb.fn.Call(in)

		cb = newCallback(func(i [10]string) {
			if len(i) != len(strings) || cap(i) != cap(strings) {
				t.Error("unmarshal [10]string incorrect")
			}
			for j := range i {
				if i[j] != strings[j] {
					t.Error("unmarshal [10]string", j, "incorrect")
				}
			}
		})
		in, err = (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil)
		if err != nil {
			t.Error(err.Error())
			return
		}
		cb.fn.Call(in)

		cb = newCallback(func(i ...string) {
			if len(i) != len(strings) || cap(i) != cap(strings) {
				t.Error("unmarshal ...string incorrect")
			}
			for j := range i {
				if i[j] != strings[j] {
					t.Error("unmarshal ...string", j, "incorrect")
				}
			}
		})
		in, err = (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil)
		if err != nil {
			t.Error(err.Error())
			return
		}
		cb.fn.CallSlice(in)
	}
	{
		data, err := msgp.AppendIntf(nil, []interface{}{bytess})
		if err != nil {
			t.Fatal(err.Error())
		}
		cb := newCallback(func(i [][]byte) {
			if len(i) != len(bytess) || cap(i) != cap(bytess) {
				t.Error("unmarshal [][]byte incorrect")
			}
			for j := range i {
				if !bytes.Equal(i[j], bytess[j]) {
					t.Error("unmarshal [][]byte", j, "incorrect")
				}
			}
		})
		in, err := (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil)
		if err != nil {
			t.Error(err.Error())
			return
		}
		cb.fn.Call(in)

		cb = newCallback(func(i [10][]byte) {
			if len(i) != len(bytess) || cap(i) != cap(bytess) {
				t.Error("unmarshal [10][]byte incorrect")
			}
			for j := range i {
				if !bytes.Equal(i[j], bytess[j]) {
					t.Error("unmarshal [10][]byte", j, "incorrect")
				}
			}
		})
		in, err = (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil)
		if err != nil {
			t.Error(err.Error())
			return
		}
		cb.fn.Call(in)

		cb = newCallback(func(i ...[]byte) {
			if len(i) != len(bytess) || cap(i) != cap(bytess) {
				t.Error("unmarshal ...[]byte incorrect")
			}
			for j := range i {
				if !bytes.Equal(i[j], bytess[j]) {
					t.Error("unmarshal ...[]byte", j, "incorrect")
				}
			}
		})
		in, err = (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil)
		if err != nil {
			t.Error(err.Error())
			return
		}
		cb.fn.CallSlice(in)
	}
	{
		data, err := msgp.AppendIntf(nil, []interface{}{foos})
		if err != nil {
			t.Fatal(err.Error())
		}
		cb := newCallback(func(i []foobar) {
			if len(i) != len(foos) || cap(i) != cap(foos) {
				t.Error("unmarshal []foobar incorrect")
			}
			for j := range i {
				if i[j].Foo != foos[j].Foo {
					t.Error("unmarshal []foobar", j, "incorrect")
				}
			}
		})
		in, err := (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil)
		if err != nil {
			t.Error(err.Error())
			return
		}
		cb.fn.Call(in)

		cb = newCallback(func(i [4]foobar) {
			if len(i) != len(foos) || cap(i) != cap(foos) {
				t.Error("unmarshal [4]foobar incorrect")
			}
			for j := range i {
				if i[j].Foo != foos[j].Foo {
					t.Error("unmarshal [4]foobar", j, "incorrect")
				}
			}
		})
		in, err = (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil)
		if err != nil {
			t.Error(err.Error())
			return
		}
		cb.fn.Call(in)

		cb = newCallback(func(i ...foobar) {
			if len(i) != len(foos) || cap(i) != cap(foos) {
				t.Error("unmarshal ...foobar incorrect")
			}
			for j := range i {
				if i[j].Foo != foos[j].Foo {
					t.Error("unmarshal ...foobar", j, "incorrect")
				}
			}
		})
		in, err = (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil)
		if err != nil {
			t.Error(err.Error())
			return
		}
		cb.fn.CallSlice(in)
	}
	{
		data, err := msgp.AppendIntf(nil, []interface{}{foos[0]})
		if err != nil {
			t.Fatal(err.Error())
		}

		cb := newCallback(func(i map[string]string) {
			if i["foo"] != "a" {
				t.Error("unmarshal map[string]string incorrect", i)
			}
		})
		if in, err := (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil); err != nil {
			t.Error(err.Error())
		} else {
			cb.fn.Call(in)
		}

		data, err = msgp.AppendIntf(nil, []interface{}{map[string]interface{}{"foo": 0x1234}})
		if err != nil {
			t.Fatal(err.Error())
		}

		cb = newCallback(func(i map[string]int64) {
			if i["foo"] != 0x1234 {
				t.Error("unmarshal map[string]int64 incorrect", i)
			}
		})
		if in, err := (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil); err != nil {
			t.Error(err.Error())
		} else {
			cb.fn.Call(in)
		}

		data, err = msgp.AppendIntf(nil, []interface{}{map[string]interface{}{"foo": float64(1.234)}})
		if err != nil {
			t.Fatal(err.Error())
		}

		cb = newCallback(func(i map[string]float64) {
			if i["foo"] != 1.234 {
				t.Error("unmarshal map[string]float64 incorrect", i)
			}
		})
		if in, err := (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil); err != nil {
			t.Error(err.Error())
		} else {
			cb.fn.Call(in)
		}

		data, err = msgp.AppendIntf(nil, []interface{}{map[string]interface{}{"foo": []byte{1, 2, 3, 4}}})
		if err != nil {
			t.Fatal(err.Error())
		}

		cb = newCallback(func(i map[string][]byte) {
			if !bytes.Equal(i["foo"], []byte{1, 2, 3, 4}) {
				t.Error("unmarshal map[string][]byte incorrect", i)
			}
		})
		if in, err := (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil); err != nil {
			t.Error(err.Error())
		} else {
			cb.fn.Call(in)
		}
	}
	{
		data, err := msgp.AppendIntf(nil, []interface{}{foos[0]})
		if err != nil {
			t.Fatal(err.Error())
		}

		cb := newCallback(func(i unsafe.Pointer) {})
		if _, err := (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil); err == nil {
			t.Error("unmarshal should return error")
		}

		cb = newCallback(func(i msgp.Unmarshaler) {})
		if _, err = (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil); err == nil {
			t.Error("unmarshal should return error")
		}

		cb = newCallback(func(i struct{}) {})
		if _, err = (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil); err == nil {
			t.Error("unmarshal should return error")
		}

		cb = newCallback(func(i chan byte) {})
		if _, err = (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil); err == nil {
			t.Error("unmarshal should return error")
		}

		cb = newCallback(func(i func()) {})
		if _, err = (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil); err == nil {
			t.Error("unmarshal should return error")
		}
	}
}

func TestMsgpackUnmarshalArgs(t *testing.T) {
	data, err := msgp.AppendIntf(nil, []interface{}{
		int(100), uint32(200), int64(300), []byte{1, 2, 3, 4}, "string", &foobar{Foo: "bar"},
		complex64(400), &msgp.RawExtension{Type: 99, Data: []byte{1, 0, 1, 0, 1}},
		map[string]interface{}{"a": "value"},
	})

	if err != nil {
		t.Fatal(err.Error())
	}

	cb := newCallback(func(i int, j uint32, k int64, b []byte, s string, foo foobar, c complex64, e extfoo, m map[string]interface{}) {
		if i != 100 || j != 200 || k != 300 || !bytes.Equal(b, []byte{1, 2, 3, 4}) || s != "string" || c != complex64(400) {
			t.Error("arguments incorrect")
		}

		if foo.Foo != "bar" {
			t.Error("msgp.Unmarshaler incorrect")
		}

		if v, ok := m["a"]; !ok {
			t.Error("argument: map incorrect")
		} else if ss, ok := v.(string); !ok {
			t.Error("argument: map value type incorrect")
		} else if ss != "value" {
			t.Error("argument: map value incorrect")
		}

		if !bytes.Equal(e.Data, []byte{1, 0, 1, 0, 1}) {
			t.Error("argument: extension incorrect")
		}
	})

	in, err := (msgpackDecoder{}).UnmarshalArgs(cb.args, data, nil)
	if err != nil {
		t.Error(err.Error())
		return
	}
	cb.fn.Call(in)
}

type extfoo struct{ msgp.RawExtension }

func (extfoo) ExtensionType() int8 { return 99 }

type foobar struct {
	Foo string `msg:"foo"`
}

// MarshalMsg implements msgp.Marshaler
func (z foobar) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "foo"
	o = append(o, 0x81, 0xa3, 0x66, 0x6f, 0x6f)
	o = msgp.AppendString(o, z.Foo)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *foobar) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "foo":
			z.Foo, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z foobar) Msgsize() (s int) {
	s = 1 + 4 + msgp.StringPrefixSize + len(z.Foo)
	return
}
