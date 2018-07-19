package engine

import (
	"sync"
	"testing"
	"time"
)

func TestPollingConn(t *testing.T) {
	var err error
	conn := newPollingConn(0, "", "")
	defer conn.Close()

	if err = conn.SetReadDeadline(time.Now().Add(time.Millisecond * 10)); err != nil {
		t.Error(err.Error())
	}
	_, err = conn.ReadPacket() // in
	if err != ErrPollingConnReadTimeout {
		t.Error("should be timeout, but:", err)
	}
	if err = conn.SetWriteDeadline(time.Now().Add(time.Millisecond * 10)); err != nil {
		t.Error(err.Error())
	}
	err = conn.WritePacket(&Packet{msgType: MessageTypeString, pktType: PacketTypeNoop})
	if err != ErrPollingConnWriteTimeout {
		t.Error("should be timeout, but:", err)
	}

	if err = conn.SetReadDeadline(time.Time{}); err != nil {
		t.Error(err.Error())
	}
	if err = conn.SetWriteDeadline(time.Time{}); err != nil {
		t.Error(err.Error())
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := conn.ReadPacket()
		if err != ErrPollingConnPaused {
			t.Error("read should be paused, but:", err)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := conn.WritePacket(&Packet{msgType: MessageTypeString, pktType: PacketTypeNoop})
		if err != ErrPollingConnPaused {
			t.Error("write should be paused, but:", err)
		}
	}()
	if err = conn.Pause(); err != nil {
		t.Error(err.Error())
	}
	wg.Wait()

	if err = conn.Resume(); err != nil {
		t.Error(err.Error())
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := conn.ReadPacket()
		if err != ErrPollingConnClosed {
			t.Error("read should be closed: but", err)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := conn.WritePacket(&Packet{msgType: MessageTypeString, pktType: PacketTypeNoop})
		if err != ErrPollingConnClosed {
			t.Error("write should be closed: but", err)
		}
	}()
	conn.Close()
	wg.Wait()
}
