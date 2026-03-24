#!/usr/bin/env bun

// #MISE description="Build Tauri Windows app: sync code to VM, build, pull artifacts back"
// #MISE alias="vm-build"
// #MISE depends=["vm:sync"]

import { $ } from "bun";
import { mkdirSync } from "fs";
import { join } from "path";
import {
  PROJECT_DIR, PROJECT_NAME, LOGDIR, VM_USER, VM_HOME,
  ensureSshpass, ssh, scp, info, ok, die, log, timestamp,
} from "../_lib.ts";

const LOG = "vm-build.log";
log(`── ${timestamp()} ──`, LOG);

const vmProjectDir = `${VM_HOME}\\${PROJECT_NAME}`;
const artifactsDir = join(PROJECT_DIR, ".build", "windows");

await ensureSshpass();

// ── Build inside VM ───────────────────────────────────────────────────────

info("Installing tools inside VM (mise install)...", LOG);
const install = await ssh(`cd "${vmProjectDir}" && mise trust && mise install`);
if (install.exitCode !== 0) die("mise install failed inside VM");
ok("Tools installed", LOG);

info("Building Tauri Windows app inside VM (this takes a while on first run)...", LOG);
const build = await ssh(`cd "${vmProjectDir}" && mise run build`);
if (build.exitCode !== 0) die("Build failed inside VM");
ok("Build complete", LOG);

// ── Pull artifacts back ───────────────────────────────────────────────────

info("Pulling artifacts...", LOG);
mkdirSync(artifactsDir, { recursive: true });

await ssh(
  `cd "${vmProjectDir}\\src-tauri\\target\\release\\bundle" && tar -czf "%USERPROFILE%\\artifacts.tar.gz" .`,
);
await scp(`${VM_USER}@127.0.0.1:artifacts.tar.gz`, join(artifactsDir, "artifacts.tar.gz"));
await $`tar -xzf ${join(artifactsDir, "artifacts.tar.gz")} -C ${artifactsDir}`;
await $`rm -f ${join(artifactsDir, "artifacts.tar.gz")}`.nothrow();
await ssh(`del "%USERPROFILE%\\artifacts.tar.gz"`).catch(() => {});

log("", LOG);
ok(`Windows build artifacts in ${artifactsDir}:`, LOG);

const glob = new Bun.Glob("**/*.{msi,exe}");
for await (const path of glob.scan(artifactsDir)) {
  const file = Bun.file(join(artifactsDir, path));
  const size = (file.size / 1024 / 1024).toFixed(1);
  log(`  ${path} (${size} MB)`, LOG);
}
