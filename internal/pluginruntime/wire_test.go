package pluginruntime

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
)

func TestReadFrame_TooLarge(t *testing.T) {
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.BigEndian, uint32(maxFrameSize+1))
	_, err := readFrame(&buf)
	if err == nil {
		t.Fatal("expected error for oversized length prefix")
	}
}

func TestWriteReadRoundTripWireReq(t *testing.T) {
	var w bytes.Buffer
	req := wireReq{Op: OpHealth, Payload: []byte(`{"a":1}`)}
	if err := writeFrame(&w, req); err != nil {
		t.Fatal(err)
	}
	raw, err := readFrame(&w)
	if err != nil {
		t.Fatal(err)
	}
	var got wireReq
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Op != OpHealth {
		t.Fatalf("op %q", got.Op)
	}
}
