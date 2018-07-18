package engine

import (
	"sync"
	"testing"
	"time"
)

func TestPollingConn(t *testing.T) {
	var err error
	conn := NewPollingConn(0, "", "")
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(time.Millisecond * 10))
	_, err = conn.ReadPacket() // in
	if err != ErrPollingConnReadTimeout {
		t.Error("should be timeout, but:", err)
	}
	conn.SetWriteDeadline(time.Now().Add(time.Millisecond * 10))
	err = conn.WritePacket(&Packet{msgType: MessageTypeString, pktType: PacketTypeNoop})
	if err != ErrPollingConnWriteTimeout {
		t.Error("should be timeout, but:", err)
	}

	conn.SetReadDeadline(time.Time{})
	conn.SetWriteDeadline(time.Time{})

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
	conn.Pause()
	wg.Wait()

	conn.Resume()
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
