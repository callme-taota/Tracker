package pluginruntime

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestClientHandshakeAndProcess(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	addr := ln.Addr().String()
	done := make(chan struct{})
	go func() {
		defer close(done)
		c, err := ln.Accept()
		if err != nil {
			t.Error(err)
			return
		}
		defer c.Close()
		_ = ln.Close()
		servePluginConn(t, c)
	}()

	cli, err := Dial(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer closeClientAndWait(t, cli, done)

	hs, err := cli.Handshake(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if hs.PluginID != "test_ext" {
		t.Fatalf("plugin_id %q", hs.PluginID)
	}
	var man map[string]interface{}
	if err := json.Unmarshal(hs.Manifest, &man); err != nil {
		t.Fatal(err)
	}
	if man["kind"] != "processor" {
		t.Fatalf("manifest kind %v", man["kind"])
	}

	item := map[string]string{"title": "hi", "content": "c"}
	itemJ, _ := json.Marshal(item)
	raw, err := cli.ExecuteProcess(context.Background(), itemJ, map[string]interface{}{"k": 1})
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out["title"] != "HI" {
		t.Fatalf("got %v", out["title"])
	}
}

func TestClientHandshake_RemoteError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	addr := ln.Addr().String()
	done := make(chan struct{})
	go func() {
		defer close(done)
		c, err := ln.Accept()
		if err != nil {
			t.Error(err)
			return
		}
		defer c.Close()
		_ = ln.Close()
		for {
			_, err := readFrame(c)
			if err == io.EOF {
				return
			}
			if err != nil {
				t.Error(err)
				return
			}
			_ = writeFrame(c, wireResp{OK: false, ErrCode: "test_err", ErrMsg: "intentional failure"})
		}
	}()

	cli, err := Dial(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer closeClientAndWait(t, cli, done)

	_, err = cli.Handshake(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "intentional failure") {
		t.Fatalf("got %v", err)
	}
}

func closeClientAndWait(t *testing.T, cli *Client, done <-chan struct{}) {
	t.Helper()
	if cli != nil {
		_ = cli.Close()
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("test plugin server did not exit")
	}
}

func servePluginConn(t *testing.T, c net.Conn) {
	manifest := json.RawMessage(`{"id":"test_ext","version":"1","kind":"processor","input_formats":["tracker.item.v1"],"output_formats":["tracker.item.v1"]}`)
	for {
		body, err := readFrame(c)
		if err == io.EOF {
			return
		}
		if err != nil {
			t.Error(err)
			return
		}
		var req wireReq
		if err := json.Unmarshal(body, &req); err != nil {
			t.Error(err)
			return
		}
		switch req.Op {
		case OpHandshake:
			var m interface{}
			_ = json.Unmarshal(manifest, &m)
			resp := map[string]interface{}{
				"plugin_id": "test_ext",
				"version":   "1",
				"manifest":  m,
			}
			b, _ := json.Marshal(resp)
			_ = writeFrame(c, wireResp{OK: true, Payload: b})
		case OpExecuteProcess:
			var in struct {
				Item json.RawMessage `json:"item"`
			}
			_ = json.Unmarshal(req.Payload, &in)
			var it map[string]string
			_ = json.Unmarshal(in.Item, &it)
			if it["title"] != "" {
				it["title"] = "HI"
			}
			out, _ := json.Marshal(it)
			p, _ := json.Marshal(map[string]json.RawMessage{"item": out})
			_ = writeFrame(c, wireResp{OK: true, Payload: p})
		default:
			_ = writeFrame(c, wireResp{OK: true, Payload: json.RawMessage(`{}`)})
		}
	}
}
