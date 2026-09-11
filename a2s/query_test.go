package a2s

import (
	"encoding/binary"
	"net"
	"strings"
	"testing"
	"time"
)

func buildInfoResponse() []byte {
	var b []byte
	w := func(v byte) { b = append(b, v) }
	s := func(v string) { b = append(b, v...); b = append(b, 0) }

	b = append(b, 0xFF, 0xFF, 0xFF, 0xFF, respInfo)
	w(17)          // protocol
	s("WOC Chernobyl") // name
	s("chernarusplus")
	s("dayz")
	s("DayZ")
	var appid [2]byte
	binary.LittleEndian.PutUint16(appid[:], 36684) // truncated 221100&0xFFFF
	b = append(b, appid[:]...)
	w(12) // players
	w(60) // max players
	w(0)  // bots
	w('d')
	w('l')
	w(0)
	w(1) // VAC
	s("0.3.1.2") // version
	flags := []byte{0x80 | 0x10 | 0x20 | 0x01}
	b = append(b, flags...)
	var port [2]byte
	binary.LittleEndian.PutUint16(port[:], 2402)
	b = append(b, port[:]...)
	var sid [8]byte
	binary.LittleEndian.PutUint64(sid[:], 90071996650678277)
	b = append(b, sid[:]...)
	s("mod=@CF;@WOC_RP")
	var gid [8]byte
	binary.LittleEndian.PutUint64(gid[:], 221100)
	b = append(b, gid[:]...)
	return b
}

// fakeServer отвечает на A2S_INFO.
func fakeServer(t *testing.T) (string, string) {
	udp, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 1024)
		for {
			n, addr, err := udp.ReadFrom(buf)
			if err != nil {
				return
			}
			if n < 5 || buf[4] != headerA2SInfo {
				continue
			}
			udp.WriteTo(buildInfoResponse(), addr)
			return
		}
	}()
	t.Cleanup(func() { udp.Close(); <-done })
	return udp.LocalAddr().String(), ""
}

func TestInfo(t *testing.T) {
	addr, _ := fakeServer(t)
	c := New()
	c.Timeout = time.Second
	info, ping, err := c.Info(addr)
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if !strings.Contains(info.Name, "WOC") {
		t.Errorf("name = %q", info.Name)
	}
	if info.MaxPlayers != 60 {
		t.Errorf("maxPlayers = %d", info.MaxPlayers)
	}
	if info.Players != 12 {
		t.Errorf("players = %d", info.Players)
	}
	if info.GameID != 221100 {
		t.Errorf("gameID = %d", info.GameID)
	}
	if !strings.Contains(info.Keywords, "@CF") {
		t.Errorf("keywords = %q", info.Keywords)
	}
	if info.Map != "chernarusplus" {
		t.Errorf("map = %q", info.Map)
	}
	if ping < 0 {
		t.Errorf("ping = %v", ping)
	}
}