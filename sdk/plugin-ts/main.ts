/**
 * Minimal Tracker external plugin (TypeScript, Deno).
 * Run: deno run -A main.ts
 * YAML command example: ["deno","run","-A","/abs/path/main.ts"]
 */

const maxFrame = 32 << 20;

async function readFull(conn: Deno.Conn, buf: Uint8Array) {
  let o = 0;
  while (o < buf.length) {
    const n = await conn.read(buf.subarray(o));
    if (n === null) throw new Error("unexpected EOF");
    o += n;
  }
}

async function readFrame(conn: Deno.Conn): Promise<Uint8Array | null> {
  const h = new Uint8Array(4);
  try {
    await readFull(conn, h);
  } catch {
    return null;
  }
  const len = (h[0] << 24) | (h[1] << 16) | (h[2] << 8) | h[3];
  if (len > maxFrame) throw new Error("frame too large");
  const body = new Uint8Array(len);
  await readFull(conn, body);
  return body;
}

async function writeFrame(conn: Deno.Conn, obj: unknown) {
  const b = new TextEncoder().encode(JSON.stringify(obj));
  if (b.length > maxFrame) throw new Error("payload too large");
  const lb = new Uint8Array(4);
  lb[0] = (b.length >>> 24) & 0xff;
  lb[1] = (b.length >>> 16) & 0xff;
  lb[2] = (b.length >>> 8) & 0xff;
  lb[3] = b.length & 0xff;
  let o = 0;
  while (o < lb.length) {
    o += await conn.write(lb.subarray(o));
  }
  o = 0;
  while (o < b.length) {
    o += await conn.write(b.subarray(o));
  }
}

const manifest = {
  id: "ts_demo",
  version: "1.0",
  kind: "processor",
  display_name: "TS demo (external)",
  input_formats: ["tracker.item.v1"],
  output_formats: ["tracker.item.v1"],
};

function handle(op: string, payload: Record<string, unknown>): unknown {
  switch (op) {
    case "handshake":
      return {
        plugin_id: manifest.id,
        version: manifest.version,
        manifest,
      };
    case "health":
    case "ready":
      return { status: "ok" };
    case "init":
    case "test_config":
      return {};
    case "execute_source":
      return { items: [] };
    case "execute_process": {
      const item = (payload.item ?? {}) as Record<string, unknown>;
      const title = String(item.title ?? "");
      return {
        item: { ...item, title: title.toUpperCase() },
      };
    }
    case "execute_dispatch":
      return {};
    default:
      return {};
  }
}

const listener = Deno.listen({ hostname: "127.0.0.1", port: 0 });
const a = listener.addr as Deno.NetAddr;
const line = JSON.stringify({ listen_addr: `${a.hostname}:${a.port}` });
await Deno.stdout.write(new TextEncoder().encode(line + "\n"));

const conn = await listener.accept();
try {
  while (true) {
    const raw = await readFrame(conn);
    if (raw === null) break;
    const req = JSON.parse(new TextDecoder().decode(raw)) as {
      op: string;
      payload?: Record<string, unknown>;
    };
    try {
      const pay = req.payload ?? {};
      const out = handle(req.op, pay);
      await writeFrame(conn, { ok: true, payload: out });
    } catch (e) {
      await writeFrame(conn, {
        ok: false,
        err_code: "error",
        err_msg: String(e),
      });
    }
  }
} finally {
  listener.close();
  conn.close();
}
