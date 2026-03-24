// WebDriver screenshot infrastructure for Tauri apps.
// Used by screenshot.ts — not a user-facing task.

import { sleep } from "bun";
import { existsSync, readFileSync, unlinkSync, readdirSync } from "fs";
import { join } from "path";
import { log, info } from "./_lib.ts";

// ── Types ──────────────────────────────────────────────────────────────────

export type Session = {
  url: string;
  sessionId: string;
  cleanup: () => void;
};

// ── Internals ──────────────────────────────────────────────────────────────

const procs: Bun.Subprocess[] = [];

function spawn(cmd: string[], env?: Record<string, string | undefined>): void {
  procs.push(Bun.spawn(cmd, { stdout: "pipe", stderr: "pipe", env }));
}

export function cleanup() {
  for (const p of procs) { try { p.kill(); } catch {} }
  // Remove stale single-instance sockets left by tauri-plugin-single-instance.
  // Without this, the next launch silently exits thinking an instance is running.
  cleanSingleInstanceSocket();
}

function cleanSingleInstanceSocket() {
  try {
    const tmpdir = "/tmp";
    for (const f of readdirSync(tmpdir)) {
      if (f.endsWith("_si.sock")) {
        try { unlinkSync(join(tmpdir, f)); } catch {}
      }
    }
  } catch {}
}

async function poll(label: string, fn: () => Promise<boolean>, ms: number, interval = 500) {
  const end = Date.now() + ms;
  while (Date.now() < end) {
    if (await fn()) return;
    await sleep(interval);
  }
  throw new Error(`${label}: timed out after ${ms / 1000}s`);
}

async function wdPost(base: string, path: string, body?: unknown): Promise<any> {
  const res = await fetch(`${base}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) throw new Error(`POST ${path} → ${res.status}`);
  return res.json();
}

// ── Public ─────────────────────────────────────────────────────────────────

/**
 * Build the Tauri app with WebDriver support, launch it, and return
 * a live session. Caller must call cleanup() when done.
 */
export async function startSession(projectRoot: string, port = 4444): Promise<Session> {
  if (!Bun.which("tauri-webdriver")) {
    throw new Error("tauri-webdriver not found. Install: cargo install tauri-webdriver --locked");
  }

  const conf = JSON.parse(readFileSync(join(projectRoot, "src-tauri/tauri.conf.json"), "utf-8"));
  const appName = conf.productName ?? "app";
  const manifest = join(projectRoot, "src-tauri/Cargo.toml");
  const binary = join(projectRoot, "src-tauri/target/debug", appName);

  // Build
  info("Building with --features webdriver...");
  const build = Bun.spawnSync(
    ["cargo", "build", "--manifest-path", manifest, "--features", "webdriver"],
    { stdout: "inherit", stderr: "inherit" },
  );
  if (build.exitCode !== 0) throw new Error("cargo build failed");
  if (!existsSync(binary)) throw new Error(`Binary not found: ${binary}`);
  log("");

  // Clean stale single-instance sockets before launching
  cleanSingleInstanceSocket();

  // Launch proxy + app
  const url = `http://127.0.0.1:${port}`;
  info("Starting WebDriver...");
  spawn(["tauri-webdriver", "--port", String(port)]);
  spawn([binary], { ...process.env, TAURI_WEBVIEW_AUTOMATION: "true" });

  // Poll for session — this waits for both proxy and app plugin to be ready
  let sessionId: string | null = null;
  await poll("Session", async () => {
    try {
      const r = await wdPost(url, "/session", {
        capabilities: { alwaysMatch: { "tauri:options": { application: binary } } },
      });
      sessionId = r.value?.sessionId ?? r.sessionId;
      return !!sessionId;
    } catch { return false; }
  }, 45_000, 2000);
  if (!sessionId) throw new Error("Could not create session");
  log(`  session ${sessionId}`);

  // Wait for page load
  await poll("Page", async () => {
    try {
      const r = await wdPost(url, `/session/${sessionId}/execute/sync`, {
        script: "return document.readyState === 'complete'", args: [],
      });
      return r.value === true;
    } catch { return false; }
  }, 10_000);
  log("  ready\n");

  process.on("SIGINT", () => { cleanup(); process.exit(130); });
  process.on("SIGTERM", () => { cleanup(); process.exit(143); });

  return { url, sessionId, cleanup };
}
