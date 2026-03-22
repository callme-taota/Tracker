package pluginruntime

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

const maxFrameSize = 32 << 20 // 32 MiB

// wireReq is one JSON-RPC-style envelope over a length-prefixed frame.
type wireReq struct {
	Op      string          `json:"op"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type wireResp struct {
	OK      bool            `json:"ok"`
	Payload json.RawMessage `json:"payload,omitempty"`
	ErrCode string          `json:"err_code,omitempty"`
	ErrMsg  string          `json:"err_msg,omitempty"`
}

func readFrame(r io.Reader) ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(lenBuf[:])
	if n > maxFrameSize {
		return nil, fmt.Errorf("pluginruntime: frame %d exceeds max %d", n, maxFrameSize)
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
	if len(b) > maxFrameSize {
		return fmt.Errorf("pluginruntime: payload exceeds max frame size")
	}
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(b)))
	if _, err := w.Write(lenBuf[:]); err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}
