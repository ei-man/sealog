import { logs } from './stores/logs.svelte';
import type { ClientCommand, ServerMessage } from './types';

export type LogConnection = {
  send: (command: ClientCommand) => void;
  disconnect: () => void;
};

export function connectLogs(): LogConnection {
  let socket: WebSocket | null = null;
  let reconnectTimer: number | null = null;
  let attempts = 0;
  let stopped = false;
  const pending: ClientCommand[] = [];

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
      logs.setConnected(true);
      while (pending.length && socket?.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify(pending.shift()));
      }
    });

    socket.addEventListener('message', (event) => {
      const message = JSON.parse(event.data) as ServerMessage;
      if (message.type === 'fields') {
        logs.setFields(message.fields);
      } else if (message.type === 'line') {
        logs.addLine(message.line);
      } else if (message.type === 'source') {
        logs.setSource(message.source);
      } else if (message.type === 'timeRange') {
        logs.setTimeRange(message.timeRange);
      } else if (message.type === 'results.reset') {
        logs.resetResults(message.total);
      } else if (message.type === 'results.stats') {
        logs.setTotal(message.total);
      } else if (message.type === 'query.error') {
        const position = message.error.position >= 0 ? ` at position ${message.error.position + 1}` : '';
        logs.setQueryError(`${message.error.message}${position}`);
      } else if (message.type === 'azure.resources') {
        logs.setAzureResources(message.resources, message.errorMessage ?? '');
      } else if (message.type === 'loading') {
        logs.setLoading(message.loading);
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

  function send(command: ClientCommand) {
    if (socket?.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify(command));
      return;
    }
    pending.push(command);
  }

  return {
    send,
    disconnect: () => {
      stopped = true;
      clearReconnectTimer();
      socket?.close();
    }
  };
}
