// PASETO token storage.
//
// Two backends behind one async interface:
//   - Tauri desktop: the token is written through a Rust command into
//     tauri-plugin-store (OS app-data dir), so page script cannot read the file
//     and it survives reloads.
//   - Browser: sessionStorage (in-memory for the tab's lifetime), NOT
//     localStorage — an XSS can still call the API while the tab is open, but the
//     token no longer persists on disk for a later injection to steal.
//
// The `getToken/setToken/clearToken` functions are the single seam; callers
// await them and never touch storage directly.
import { invoke } from "@tauri-apps/api/core";

const KEY = "veemon.token";

function isTauri(): boolean {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}

export async function getToken(): Promise<string | null> {
  if (isTauri()) return invoke<string | null>("get_token");
  return sessionStorage.getItem(KEY);
}

export async function setToken(token: string): Promise<void> {
  if (isTauri()) {
    await invoke("set_token", { token });
    return;
  }
  sessionStorage.setItem(KEY, token);
}

export async function clearToken(): Promise<void> {
  if (isTauri()) {
    await invoke("clear_token");
    return;
  }
  sessionStorage.removeItem(KEY);
}
