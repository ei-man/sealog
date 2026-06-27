import './app.css';
import App from './App.svelte';
import { mount } from 'svelte';

function describeError(error: unknown): Record<string, unknown> {
  if (error instanceof Error) {
    return {
      name: error.name,
      message: error.message,
      stack: error.stack,
      cause: error.cause
    };
  }

  return {
    value: error
  };
}

window.addEventListener('error', (event) => {
  console.error('[Sealog runtime error]', {
    message: event.message,
    source: event.filename,
    line: event.lineno,
    column: event.colno,
    error: describeError(event.error)
  });
});

window.addEventListener('unhandledrejection', (event) => {
  console.error('[Sealog unhandled promise rejection]', {
    reason: describeError(event.reason)
  });
});

const app = mount(App, {
  target: document.getElementById('app') as HTMLElement
});

export default app;
