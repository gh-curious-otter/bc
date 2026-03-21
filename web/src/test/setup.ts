import '@testing-library/jest-dom/vitest';
import { vi } from 'vitest';

globalThis.fetch = vi.fn();

class FakeEventSource {
  onopen: (() => void) | null = null;
  onmessage: ((e: MessageEvent) => void) | null = null;
  onerror: (() => void) | null = null;
  close() {}
}
globalThis.EventSource = FakeEventSource as unknown as typeof EventSource;
