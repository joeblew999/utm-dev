#!/usr/bin/env bun

// Project-specific screenshots for tauri-basic.
// Called by `mise run screenshot` with WEBDRIVER_URL + WEBDRIVER_SESSION set.
// Uses plain fetch() — no utm-dev imports needed.

import { readFileSync, writeFileSync, mkdirSync } from "fs";
import { join } from "path";

const URL = process.env.WEBDRIVER_URL!;
const SID = process.env.WEBDRIVER_SESSION!;
if (!URL || !SID) { console.error("Run via: mise run screenshot"); process.exit(1); }

const ROOT = process.cwd();
const OUT = join(ROOT, "screenshots");

// ── WebDriver helpers ──────────────────────────────────────────────────────

async function exec(script: string): Promise<any> {
  const r = await fetch(`${URL}/session/${SID}/execute/sync`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ script, args: [] }),
  });
  return (await r.json() as any).value;
}

async function waitFor(label: string, script: string, ms = 5000) {
  const end = Date.now() + ms;
  while (Date.now() < end) {
    try { if (await exec(script) === true) return; } catch {}
    await Bun.sleep(100);
  }
  throw new Error(`${label}: timed out`);
}

async function capture(filepath: string) {
  const r = await fetch(`${URL}/session/${SID}/screenshot`);
  const { value } = await r.json() as { value: string };
  writeFileSync(filepath, Buffer.from(value, "base64"));
}

// ── Discover tabs from HTML ────────────────────────────────────────────────

const html = readFileSync(join(ROOT, "ui/index.html"), "utf-8");
const allTabs = [...html.matchAll(/data-panel="([^"]+)"/g)].map((m) => m[1]);

const requested = process.argv.slice(2).filter((a) => !a.startsWith("-"));
const tabs = requested.length > 0 ? requested : allTabs;
const bad = tabs.filter((t) => !allTabs.includes(t));
if (bad.length) { console.error(`Unknown: ${bad.join(", ")} | Available: ${allTabs.join(", ")}`); process.exit(1); }

// ── Capture each tab ───────────────────────────────────────────────────────

mkdirSync(OUT, { recursive: true });
await waitFor("Nav", "return document.querySelectorAll('nav button[data-panel]').length > 0");

let ok = 0;
for (const tab of tabs) {
  try {
    await exec(`document.querySelector('nav button[data-panel="${tab}"]')?.click()`);
    await waitFor(tab, `return document.getElementById('panel-${tab}')?.classList.contains('active')`);
    await capture(join(OUT, `${tab}.png`));
    ok++;
    console.log(`  ${tab}.png`);
  } catch (e) {
    console.log(`  ${tab}: FAILED — ${e instanceof Error ? e.message : e}`);
  }
}

console.log(`\n${ok}/${tabs.length} saved to screenshots/`);
if (ok < tabs.length) process.exit(1);
