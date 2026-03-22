//! Minimal Tracker external plugin in Rust (TCP + length-prefixed JSON).
//! Build: `cargo build --release` and point `external_plugins.yaml` at the binary.

use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::io::{Read, Write};
use std::net::TcpListener;

const MAX_FRAME: usize = 32 * 1024 * 1024;

#[derive(Debug, Deserialize)]
struct WireReq {
    op: String,
    #[serde(default)]
    payload: Value,
}

#[derive(Serialize)]
struct WireResp {
    ok: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    payload: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    err_code: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    err_msg: Option<String>,
}

fn read_frame(r: &mut impl Read) -> std::io::Result<Vec<u8>> {
    let mut h = [0u8; 4];
    r.read_exact(&mut h)?;
    let len = u32::from_be_bytes(h) as usize;
    if len > MAX_FRAME {
        return Err(std::io::Error::new(
            std::io::ErrorKind::InvalidData,
            "frame too large",
        ));
    }
    let mut v = vec![0u8; len];
    r.read_exact(&mut v)?;
    Ok(v)
}

fn write_frame(w: &mut impl Write, resp: &WireResp) -> std::io::Result<()> {
    let b = serde_json::to_vec(resp)?;
    if b.len() > MAX_FRAME {
        return Err(std::io::Error::new(
            std::io::ErrorKind::InvalidData,
            "payload too large",
        ));
    }
    w.write_all(&(b.len() as u32).to_be_bytes())?;
    w.write_all(&b)?;
    Ok(())
}

fn handle(op: &str, payload: &Value) -> Value {
    let manifest = json!({
        "id": "rust_demo",
        "version": "1.0",
        "kind": "processor",
        "display_name": "Rust demo (external)",
        "input_formats": ["tracker.item.v1"],
        "output_formats": ["tracker.item.v1"],
    });
    match op {
        "handshake" => json!({
            "plugin_id": "rust_demo",
            "version": "1.0",
            "manifest": manifest,
        }),
        "health" | "ready" => json!({"status": "ok"}),
        "init" | "test_config" => json!({}),
        "execute_source" => json!({"items": [] as [Value; 0]}),
        "execute_process" => {
            let mut item = payload
                .get("item")
                .cloned()
                .unwrap_or(json!({}));
            if let Some(t) = item.get("title").and_then(|x| x.as_str()) {
                item["title"] = json!(t.to_uppercase());
            }
            json!({"item": item})
        }
        "execute_dispatch" => json!({}),
        _ => json!({}),
    }
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let listener = TcpListener::bind("127.0.0.1:0")?;
    let addr = listener.local_addr()?;
    println!("{{\"listen_addr\":\"{}\"}}", addr);
    let (mut stream, _) = listener.accept()?;
    loop {
        let raw = match read_frame(&mut stream) {
            Ok(r) => r,
            Err(e) if e.kind() == std::io::ErrorKind::UnexpectedEof => break,
            Err(e) => return Err(e.into()),
        };
        let req: WireReq = match serde_json::from_slice(&raw) {
            Ok(r) => r,
            Err(e) => {
                write_frame(
                    &mut stream,
                    &WireResp {
                        ok: false,
                        payload: None,
                        err_code: Some("bad_request".into()),
                        err_msg: Some(e.to_string()),
                    },
                )?;
                continue;
            }
        };
        let out = handle(&req.op, &req.payload);
        write_frame(
            &mut stream,
            &WireResp {
                ok: true,
                payload: Some(out),
                err_code: None,
                err_msg: None,
            },
        )?;
    }
    Ok(())
}
