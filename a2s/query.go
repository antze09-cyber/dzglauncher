package a2s

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"strings"
	"time"
)

const (
	headerA2SInfo   = 0x54
	headerA2SPlayer = 0x55
	headerA2SRules  = 0x56
	// header for responses
	respInfo     = 0x49 // 'I'
	respInfoShip = 0x6D // 'm'
	respPlayer   = 0x44 // 'D'
	respRules    = 0x45 // 'E'
	respChallege = 0x41 // 'A'
)

// ServerInfo содержит ответ A2S_INFO.
type ServerInfo struct {
	Protocol    byte
	Name        string
	Map         string
	Folder      string
	Game        string
	AppID       uint16
	Players     int
	MaxPlayers  int
	Bots        int
	ServerType  byte
	Environment byte
	Visibility  int
	VAC         int
	Version     string
	EDFFlags    byte
	Port        int
	SteamID     uint64
	Keywords    string
	GameID      uint64
}

// Player описывает запись A2S_PLAYER.
type Player struct {
	Index    int
	Name     string
	Score    int32
	Duration float32
}

// Client выполняет A2S UDP-запросы к серверу.
type Client struct {
	Timeout time.Duration
}

func New() *Client {
	return &Client{Timeout: 2500 * time.Millisecond}
}

// Info делает A2S_INFO запрос и возвращает информацию о сервере и ping.
func (c *Client) Info(addr string) (ServerInfo, time.Duration, error) {
	start := time.Now()
	data, err := c.query(addr, []byte("TSource Engine Query\x00"))
	if err != nil {
		return ServerInfo{}, 0, err
	}
	if len(data) >= 9 && data[4] == respChallege {
		chall := binary.LittleEndian.Uint32(data[5:9])
		req := make([]byte, 5)
		req[0] = headerA2SInfo
		binary.LittleEndian.PutUint32(req[1:5], chall)
		data, err = c.query(addr, req)
		if err != nil {
			return ServerInfo{}, 0, err
		}
	}
	if len(data) < 5 || (data[4] != respInfo && data[4] != respInfoShip) {
		return ServerInfo{}, 0, errors.New("unexpected A2S_INFO response")
	}
	return parseInfo(data[5:], data[4] == respInfoShip), time.Since(start), nil
}

// Players делает A2S_PLAYER запрос.
func (c *Client) Players(addr string) ([]Player, error) {
	data, err := c.queryChall(addr, headerA2SPlayer)
	if err != nil {
		return nil, err
	}
	if len(data) < 5 {
		return nil, errors.New("short A2S_PLAYER response")
	}
	if data[4] != respPlayer {
		return nil, fmt.Errorf("unexpected A2S_PLAYER response header 0x%02x", data[4])
	}
	r := newReader(data[5:])
	n := int(r.byte())
	players := make([]Player, 0, n)
	for i := 0; i < n && r.left() >= 6; i++ {
		p := Player{}
		p.Index = int(r.byte())
		p.Name = r.str()
		p.Score = int32(r.uint32())
		p.Duration = math.Float32frombits(r.uint32())
		players = append(players, p)
	}
	return players, nil
}

// Rules делает A2S_RULES запрос.
func (c *Client) Rules(addr string) (map[string]string, error) {
	data, err := c.queryChall(addr, headerA2SRules)
	if err != nil {
		return nil, err
	}
	if len(data) < 5 {
		return nil, errors.New("short A2S_RULES response")
	}
	if data[4] != respRules {
		return nil, fmt.Errorf("unexpected A2S_RULES response header 0x%02x", data[4])
	}
	r := newReader(data[5:])
	n := int(r.uint16())
	rules := make(map[string]string, n)
	for i := 0; i < n && r.left() > 0; i++ {
		k := r.str()
		v := r.str()
		if strings.TrimSpace(k) == "" {
			continue
		}
		rules[k] = v
	}
	return rules, nil
}

// query отправляет payload (без заголовка FF FF FF FF) и возвращает ответ.
func (c *Client) query(addr string, payload []byte) ([]byte, error) {
	conn, err := net.DialTimeout("udp", addr, c.Timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	deadline := time.Now().Add(c.Timeout)
	conn.SetDeadline(deadline)

	pkt := make([]byte, 4, 4+len(payload))
	pkt[0], pkt[1], pkt[2], pkt[3] = 0xFF, 0xFF, 0xFF, 0xFF
	pkt = append(pkt, payload...)
	if _, err := conn.Write(pkt); err != nil {
		return nil, err
	}
	buf := make([]byte, 65535)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

// queryChall отправляет запрос со значением challenge (-1) и повторяет с полученным challenge при необходимости.
func (c *Client) queryChall(addr string, header byte) ([]byte, error) {
	payload := make([]byte, 5)
	payload[0] = header
	binary.LittleEndian.PutUint32(payload[1:5], math.MaxUint32)
	data, err := c.query(addr, payload)
	if err != nil {
		return nil, err
	}
	for len(data) >= 9 && data[4] == respChallege {
		chall := binary.LittleEndian.Uint32(data[5:9])
		payload := make([]byte, 5)
		payload[0] = header
		binary.LittleEndian.PutUint32(payload[1:5], chall)
		data, err = c.query(addr, payload)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func parseInfo(b []byte, theShip bool) ServerInfo {
	r := newReader(b)
	info := ServerInfo{}
	info.Protocol = r.byte()
	info.Name = r.str()
	info.Map = r.str()
	info.Folder = r.str()
	info.Game = r.str()
	info.AppID = r.uint16()
	info.Players = int(r.byte())
	info.MaxPlayers = int(r.byte())
	info.Bots = int(r.byte())
	info.ServerType = r.byte()
	info.Environment = r.byte()
	info.Visibility = int(r.byte())
	info.VAC = int(r.byte())
	if theShip {
		r.byte() // game mode
		r.byte() // witnesses
		r.byte() // duration
	}
	info.Version = r.str()
	if r.left() >= 1 {
		info.EDFFlags = r.byte()
		if r.left() >= 2 && info.EDFFlags&0x80 != 0 {
			info.Port = int(r.uint16())
		}
		if r.left() >= 8 && info.EDFFlags&0x10 != 0 {
			info.SteamID = r.uint64()
		}
		if r.left() >= 3 && info.EDFFlags&0x40 != 0 {
			r.uint16() // spectator port
			r.str()    // spectator server name
		}
		if info.EDFFlags&0x20 != 0 {
			info.Keywords = r.str()
		}
		if r.left() >= 8 && info.EDFFlags&0x01 != 0 {
			info.GameID = r.uint64()
		}
	}
	return info
}

type reader struct {
	b []byte
	i int
}

func newReader(b []byte) *reader { return &reader{b: b} }

func (r *reader) byte() byte {
	v := r.b[r.i]
	r.i++
	return v
}

func (r *reader) uint16() uint16 {
	v := binary.LittleEndian.Uint16(r.b[r.i:])
	r.i += 2
	return v
}

func (r *reader) uint32() uint32 {
	v := binary.LittleEndian.Uint32(r.b[r.i:])
	r.i += 4
	return v
}

func (r *reader) uint64() uint64 {
	v := binary.LittleEndian.Uint64(r.b[r.i:])
	r.i += 8
	return v
}

func (r *reader) str() string {
	end := bytes.IndexByte(r.b[r.i:], 0)
	if end < 0 {
		end = len(r.b) - r.i
	}
	s := string(r.b[r.i : r.i+end])
	r.i += end + 1
	return s
}

func (r *reader) left() int { return len(r.b) - r.i }