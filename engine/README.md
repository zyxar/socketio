# engine

[![GoDoc](https://godoc.org/github.com/zyxar/socketio/engine?status.svg)](https://godoc.org/github.com/zyxar/socketio/engine)
[![license](https://img.shields.io/badge/license-New%20BSD-ff69b4.svg)](https://github.com/zyxar/socketio/blob/master/LICENSE)

[engine.io](https://github.com/socketio/engine.io) in #Go

## Install

```shell
vgo get -v github.com/zyxar/socketio/engine
```

## Example

Server:
```go
package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/zyxar/socketio/engine"
)

func main() {
	server, _ := engine.NewServer(time.Second*25, time.Second*5, func(so *engine.Socket) {
		so.On(engine.EventMessage, engine.Callback(func(typ engine.MessageType, data []byte) {
			switch typ {
			case engine.MessageTypeString:
				fmt.Fprintf(os.Stderr, "txt: %s\n", data)
			case engine.MessageTypeBinary:
				fmt.Fprintf(os.Stderr, "bin: %x\n", data)
			default:
				fmt.Fprintf(os.Stderr, "???: %x\n", data)
			}
		}))
		so.On(engine.EventPing, engine.Callback(func(_ engine.MessageType, _ []byte) {
			fmt.Fprintf(os.Stderr, "socket ping\n")
		}))
		so.On(engine.EventClose, engine.Callback(func(_ engine.MessageType, _ []byte) {
			fmt.Fprintf(os.Stderr, "socket close\n")
		}))
		so.On(engine.EventUpgrade, engine.Callback(func(_ engine.MessageType, _ []byte) {
			fmt.Fprintf(os.Stderr, "socket upgrade\n")
		}))
	})
	http.ListenAndServe(":8081", server)
}
```

Client:
```js
const url = 'ws://localhost:8081';
const eio = require('engine.io-client')(url, {});

eio.on('open', function() {
    console.log('open');
    eio.on('message', function(data) {
        if (data instanceof ArrayBuffer || data instanceof Buffer) {
            var a = new Uint8Array(data);
            console.log('receive: binary '+a.toString());
        } else {
            console.log('receive: text '+data);
        }
    });
    eio.on('upgrade', function() {
        console.log('upgrade');
    });
    eio.on('ping', function() {
        console.log('ping');
    });
    eio.on('pong', function() {
        console.log('pong');
    })
    eio.on('close', function() {
        console.log('close');
        process.exit(0);
    });
    eio.on('error', function(err) {
        console.log('error: '+err);
        process.exit(-1);
    });

    var text = 'hello';
    var ab = new ArrayBuffer(4);
    var a = new Uint8Array(ab);
    a.set([1,2,3,4]);

    console.log("sending: text "+text);
    eio.send(text);

    console.log("sending: binary 1,2,3,4");
    eio.send(ab);

    setInterval(function() {
        console.log("sending: text "+text);
        eio.send(text);

        console.log("sending: binary 1,2,3,4");
        eio.send(ab);
    }, 5*1000);
});
```
