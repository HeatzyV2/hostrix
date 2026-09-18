"use client";

import { useEffect, useRef, useState } from "react";
import { Loader2 } from "lucide-react";

function buildWsUrl(pathWithQuery: string) {
  const api = process.env.NEXT_PUBLIC_HOSTRIX_API_URL;
  if (api) {
    const u = new URL(pathWithQuery, api);
    u.protocol = u.protocol === "https:" ? "wss:" : "ws:";
    return u.toString();
  }
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}${pathWithQuery}`;
}

export function ServerConsole({ serverId }: { serverId: string }) {
  const termRef = useRef<HTMLDivElement>(null);
  const [status, setStatus] = useState<"connecting" | "connected" | "disconnected" | "error">(
    "connecting"
  );
  const [message, setMessage] = useState<string | null>(null);

  useEffect(() => {
    let disposed = false;
    let socket: WebSocket | null = null;
    let term: import("@xterm/xterm").Terminal | null = null;
    let fit: import("@xterm/addon-fit").FitAddon | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let removeResize: (() => void) | null = null;
    let dataDisposable: { dispose: () => void } | null = null;

    async function setup() {
      const [{ Terminal }, { FitAddon }] = await Promise.all([
        import("@xterm/xterm"),
        import("@xterm/addon-fit"),
      ]);
      await import("@xterm/xterm/css/xterm.css");
      if (disposed || !termRef.current) return;

      term = new Terminal({
        cursorBlink: true,
        fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
        fontSize: 13,
        theme: {
          background: "#0a0c10",
          foreground: "#e8edf5",
          cursor: "#3d9cf0",
        },
      });
      fit = new FitAddon();
      term.loadAddon(fit);
      term.open(termRef.current);
      fit.fit();

      dataDisposable = term.onData((data) => {
        if (socket?.readyState === WebSocket.OPEN) {
          socket.send(data);
        }
      });

      const onResize = () => {
        fit?.fit();
        if (socket?.readyState === WebSocket.OPEN && term) {
          socket.send(JSON.stringify({ type: "resize", cols: term.cols, rows: term.rows }));
        }
      };
      window.addEventListener("resize", onResize);
      removeResize = () => window.removeEventListener("resize", onResize);

      await connect();
    }

    async function connect() {
      if (disposed || !term) return;
      setStatus("connecting");
      setMessage(null);

      const cols = term.cols;
      const rows = term.rows;

      try {
        const ticketRes = await fetch(`/api/v1/servers/${serverId}/console`, {
          credentials: "include",
        });
        const ticketData = await ticketRes.json().catch(() => ({}));
        if (!ticketRes.ok) {
          setStatus("error");
          setMessage(ticketData.error || "Failed to get console ticket");
          scheduleReconnect();
          return;
        }

        const wsPath = `${ticketData.ws_path}?ticket=${encodeURIComponent(ticketData.ticket)}&cols=${cols}&rows=${rows}`;
        socket = new WebSocket(buildWsUrl(wsPath));
        socket.binaryType = "arraybuffer";

        socket.onopen = () => {
          if (disposed) return;
          setStatus("connected");
          term?.focus();
          socket?.send(JSON.stringify({ type: "resize", cols: term!.cols, rows: term!.rows }));
        };

        socket.onmessage = (ev) => {
          if (!term) return;
          if (typeof ev.data === "string") term.write(ev.data);
          else term.write(new Uint8Array(ev.data as ArrayBuffer));
        };

        socket.onclose = () => {
          if (disposed) return;
          setStatus("disconnected");
          scheduleReconnect();
        };

        socket.onerror = () => {
          setStatus("error");
          setMessage("WebSocket error");
        };
      } catch (e) {
        setStatus("error");
        setMessage(e instanceof Error ? e.message : "Console failed");
        scheduleReconnect();
      }
    }

    function scheduleReconnect() {
      if (disposed) return;
      reconnectTimer = setTimeout(() => {
        void connect();
      }, 2500);
    }

    void setup();

    return () => {
      disposed = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      removeResize?.();
      dataDisposable?.dispose();
      socket?.close();
      term?.dispose();
    };
  }, [serverId]);

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h2 className="font-display text-lg font-medium">Console</h2>
        <div className="flex items-center gap-2 text-xs text-ink-muted">
          {status === "connecting" ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : null}
          <span className="uppercase tracking-wide">{status}</span>
        </div>
      </div>
      {message ? <p className="text-sm text-red-400">{message}</p> : null}
      <div
        ref={termRef}
        className="h-[420px] overflow-hidden rounded-2xl border border-line bg-[#0a0c10] p-2"
      />
    </div>
  );
}
