// Package server implements the Tracker external plugin wire protocol (length-prefixed JSON over TCP).
// The process must print one line to stdout: {"listen_addr":"host:port"} then accept one connection from the host.
package server

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
)

const maxFrame = 32 << 20

type wireReq struct {
	Op      string          `json:"op"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type wireResp struct {
	OK       bool            `json:"ok"`
	Payload  json.RawMessage `json:"payload,omitempty"`
	ErrCode  string          `json:"err_code,omitempty"`
	ErrMsg   string          `json:"err_msg,omitempty"`
}

// Handler handles one logical op; return payload (marshaled to JSON) and error (mapped to wire error).
type Handler func(op string, payload json.RawMessage) (interface{}, error)

// Run listens on 127.0.0.1:0, prints listen_addr on stdout, then serves framed JSON on the first connection.
func Run(h Handler) error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer ln.Close()
	addr := ln.Addr().String()
	line, err := json.Marshal(map[string]string{"listen_addr": addr})
	if err != nil {
		return err
	}
	if _, err := os.Stdout.Write(append(line, '\n')); err != nil {
		return err
	}
	_ = os.Stdout.Sync()
	conn, err := ln.Accept()
	if err != nil {
		return err
	}
	defer conn.Close()
	return serveConn(conn, h)
}

func readFrame(r io.Reader) ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(lenBuf[:])
	if n > maxFrame {
		return nil, fmt.Errorf("frame too large")
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func writeFrame(w io.Writer, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(b) > maxFrame {
		return fmt.Errorf("payload too large")
	}
	var lb [4]byte
	binary.BigEndian.PutUint32(lb[:], uint32(len(b)))
	if _, err := w.Write(lb[:]); err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func serveConn(conn net.Conn, h Handler) error {
	for {
		raw, err := readFrame(conn)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		var req wireReq
		if err := json.Unmarshal(raw, &req); err != nil {
			_ = writeFrame(conn, wireResp{OK: false, ErrCode: "bad_request", ErrMsg: err.Error()})
			continue
		}
		out, err := h(req.Op, req.Payload)
		if err != nil {
			_ = writeFrame(conn, wireResp{OK: false, ErrCode: "error", ErrMsg: err.Error()})
			continue
		}
		var payload json.RawMessage
		if out != nil {
			b, err := json.Marshal(out)
			if err != nil {
				_ = writeFrame(conn, wireResp{OK: false, ErrCode: "marshal", ErrMsg: err.Error()})
				continue
			}
			payload = b
		}
		if err := writeFrame(conn, wireResp{OK: true, Payload: payload}); err != nil {
			return err
		}
	}
}
