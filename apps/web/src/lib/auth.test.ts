import { describe, it, expect, beforeEach } from "bun:test";

// Minimal sessionStorage stub so the browser path runs under bun.
const store = new Map<string, string>();
let localWrites = 0;
(globalThis as unknown as { sessionStorage: Storage }).sessionStorage = {
  getItem: (k: string) => store.get(k) ?? null,
  setItem: (k: string, v: string) => void store.set(k, v),
  removeItem: (k: string) => void store.delete(k),
  clear: () => store.clear(),
  key: () => null,
  length: 0,
} as Storage;
(globalThis as unknown as { localStorage: Storage }).localStorage = {
  getItem: () => null,
  setItem: () => void localWrites++,
  removeItem: () => undefined,
  clear: () => undefined,
  key: () => null,
  length: 0,
} as Storage;

const { getToken, setToken, clearToken } = await import("./auth");

describe("auth token storage (browser path)", () => {
  beforeEach(() => {
    store.clear();
    localWrites = 0;
  });

  it("returns null when no token is set", async () => {
    expect(await getToken()).toBeNull();
  });

  it("stores and reads a token", async () => {
    await setToken("abc");
    expect(await getToken()).toBe("abc");
  });

  it("clears a token", async () => {
    await setToken("abc");
    await clearToken();
    expect(await getToken()).toBeNull();
  });

  it("never writes the token to localStorage (XSS persistence)", async () => {
    await setToken("abc");
    expect(localWrites).toBe(0);
  });
});
