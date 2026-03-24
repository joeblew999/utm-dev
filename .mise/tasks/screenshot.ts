#!/usr/bin/env bun

//MISE description="Take screenshots of the running Tauri app via WebDriver"
//MISE alias="ss"

import { existsSync, writeFileSync, mkdirSync } from "fs";
import { dirname, join } from "path";
import { startSession, cleanup } from "./_screenshot.ts";
import { log, die } from "./_lib.ts";

// Find Tauri project root
let root = process.cwd();
while (!existsSync(join(root, "src-tauri/tauri.conf.json"))) {
  const parent = dirname(root);
  if (parent === root) { die("Not in a Tauri project"); }
  root = parent;
}

log("═══ Tauri Screenshot ═══\n");

try {
  const session = await startSession(root);
  const customScript = join(root, "screenshots/take.ts");

  if (existsSync(customScript)) {
    // Delegate to project-specific script with session env vars
    log("Running screenshots/take.ts\n");
    const r = Bun.spawnSync(["bun", customScript, ...process.argv.slice(2)], {
      cwd: root,
      stdout: "inherit",
      stderr: "inherit",
      env: { ...process.env, WEBDRIVER_URL: session.url, WEBDRIVER_SESSION: session.sessionId },
    });
    cleanup();
    process.exit(r.exitCode ?? 0);
  }

  // Default: single screenshot
  const out = join(root, "screenshots/app.png");
  mkdirSync(dirname(out), { recursive: true });
  log("Capturing screenshot...");
  const res = await fetch(`${session.url}/session/${session.sessionId}/screenshot`);
  const { value } = await res.json() as { value: string };
  writeFileSync(out, Buffer.from(value, "base64"));
  log("  saved to screenshots/app.png\n");
  cleanup();
} catch (e) {
  cleanup();
  die(`${e instanceof Error ? e.message : e}`);
}
