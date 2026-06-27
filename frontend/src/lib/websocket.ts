import { logs } from './stores/logs.svelte';
import type { ServerMessage } from './types';

export function connectLogs(): () => void {
  let socket: WebSocket | null = null;
  let reconnectTimer: number | null = null;
  let attempts = 0;
  let stopped = false;

  function clearReconnectTimer() {
    if (reconnectTimer !== null) {
      window.clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
  }

  function scheduleReconnect() {
    if (stopped) return;
    const delay = Math.min(1000 * 2 ** attempts, 10_000);
    attempts += 1;
    reconnectTimer = window.setTimeout(connect, delay);
  }

  function connect() {
    clearReconnectTimer();
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    socket = new WebSocket(`${protocol}//${window.location.host}/ws`);

    socket.addEventListener('open', () => {
      attempts = 0;
      logs.beginSnapshot();
      logs.setConnected(true);
    });

    socket.addEventListener('message', (event) => {
      const message = JSON.parse(event.data) as ServerMessage;
      if (message.type === 'fields') {
        logs.setFields(message.fields);
      } else if (message.type === 'line') {
        logs.addLine(message.line);
      }
    });

    socket.addEventListener('close', () => {
      logs.setConnected(false);
      scheduleReconnect();
    });

    socket.addEventListener('error', () => {
      socket?.close();
    });
  }

  connect();

  return () => {
    stopped = true;
    clearReconnectTimer();
    socket?.close();
  };
}
