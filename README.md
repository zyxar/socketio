# socketio

[![GoDoc](https://godoc.org/github.com/zyxar/socketio?status.svg)](https://godoc.org/github.com/zyxar/socketio)
[![Go Report Card](https://goreportcard.com/badge/github.com/zyxar/socketio)](https://goreportcard.com/report/github.com/zyxar/socketio)
[![license](https://img.shields.io/badge/license-New%20BSD-red.svg)](https://github.com/zyxar/socketio/blob/master/LICENSE)

[socket.io](https://socket.io/)/[engine.io](https://github.com/socketio/engine.io) in #Go


## Install

```shell
vgo get -v -u github.com/zyxar/socketio
```

## Example

Server:
```go
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/zyxar/socketio"
)

func main() {
	server, _ := socketio.NewServer(time.Second*25, time.Second*5, socketio.DefaultParser)
	server.OnConnect(func(so socketio.Socket) error {
		so.On("/", "message", func(data string) {
			log.Println(data)
		})
		so.OnError(func(nsp string, err interface{}) {
			log.Printf("socket nsp=%q, error=%v", nsp, err)
		})
		return so.Emit("/", "event", "hello world!")
	})
	http.ListenAndServe(":8081", server)
}
```
Client:
```js
const io = require('socket.io-client');
const socket = io('http://localhost:8081');
var id;

socket.on('connect', function() {
  console.log('connected');
  if (id === undefined) {
    id = setInterval(function() {
      socket.emit('message', 'hello there!')
    }, 2000);
  }
});
socket.on('event', console.log);
socket.on('disconnect', function() {
  console.log('disconnected');
  if (id) {
    clearInterval(id);
    id = undefined;
  }
});
```

### With Acknowledgements

- Server -> Client

Server:
```go
	so.Emit("/", "ack", "foo", func(msg string) {
		log.Println(msg)
	})
```
Client:
```js
  socket.on('ack', function(name, fn) {
    console.log(name);
    fn('bar');
  })
```

- Client -> Server

Server:
```go
	so.On("/", "foobar", func(data string) (string, string) {
		log.Println("foobar:", data)
		return "foo", "bar"
	})
```

Client:
```js
  socket.emit('foobar', '-wow-', function (foo, bar) {
    console.log('foobar:', foo, bar);
  });
```

### With Binary Data

Server:
```go
	so.On("/", "binary", func(data interface{}, b *socketio.Bytes) {
		log.Println(data)
		log.Printf("%x", b.Marshal())
	})
	go func() {
		b := &socketio.Bytes{}
		for {
			select {
			case <-time.After(time.Second * 2):
				t, _ := time.Now().MarshalBinary()
				b.Unmarshal(t)
				so.Emit("/", "event", "check it out!", b)
			}
		}
	}()
```

Client:
```js
  var ab = new ArrayBuffer(4);
  var a = new Uint8Array(ab);
  a.set([1,2,3,4]);

  id = setInterval(function() {
    socket.emit('binary', 'buf:', ab);
  }, 2000);

  socket.on('event', console.log);
```

#### Binary Helper for `protobuf`

```go
import (
	"github.com/golang/protobuf/proto"
)

type ProtoMessage struct {
	proto.Message
}

func (p ProtoMessage) Marshal() []byte {
	b, _ := proto.Marshal(p.Message)
	return b
}

func (p *ProtoMessage) Unmarshal(b []byte) {
	proto.Unmarshal(b, p.Message)
}
```

#### Binary Helper for `MessagePack`

```go
import (
	"github.com/tinylib/msgp/msgp"
)

type MessagePack struct {
	Message interface {
		msgp.MarshalSizer
		msgp.Unmarshaler
	}
}

func (m MessagePack) Marshal() []byte {
	b, _ := m.Message.MarshalMsg(nil)
	return b
}

func (m *MessagePack) Unmarshal(b []byte) {
	m.Message.UnmarshalMsg(b)
}
```


### Customized Namespace

Server:
```go
	so.On("/ditto", "disguise", func(msg interface{}, b socketio.Bytes) {
		log.Printf("%v: %x", msg, b.Marshal())
	})
```

Client:
```js
let ditto = io('http://localhost:8081/ditto');
ditto.emit('disguise', 'pidgey', new ArrayBuffer(8));
```

## Parser

The `encoder` and `decoder` provided by `socketio.DefaultParser` is compatible with [`socket.io-parser`](https://github.com/socketio/socket.io-parser/), complying with revision 4 of [socket.io-protocol](https://github.com/socketio/socket.io-protocol).

An `Event` or `Ack` Packet with any data satisfying `socketio.Binary` interface (e.g. `socketio.Bytes`) would be encoded as `BinaryEvent` or `BinaryAck` Packet respectively.

## TODOs

- [x] socket.io client
- [x] `namespace`
- [ ] Room
- [ ] Broadcasting
- [ ] engine.io polling client
