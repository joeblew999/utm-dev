#!/usr/bin/env bun

//MISE description="Build Tauri Windows app: sync code, build in VM, pull artifacts back"
//MISE alias="vm-build"

import { $ } from "bun";
import { mkdirSync } from "fs";
import { join } from "path";
import {
  PROJECT_DIR, PROJECT_NAME,
  getProfile, vmHome,
  ensureSshpass, checkSsh, ssh, scp, info, ok, die, log, timestamp,
} from "../_lib.ts";

const LOG = "vm-build.log";
log(`── ${timestamp()} ──`, LOG);

// Always uses the "build" VM
const profile = getProfile("build");
const home = vmHome(profile);
const vmProjectDir = `${home}\\${PROJECT_NAME}`;
const artifactsDir = join(PROJECT_DIR, ".build", "windows");

await ensureSshpass();
await checkSsh(profile);

// ── Sync code to VM ─────────────────────────────────────────────────────

info(`Syncing ${PROJECT_NAME} to build VM...`, LOG);
const tmptar = `/tmp/vm-build-${Date.now()}.tar.gz`;
try {
  await $`tar -czf ${tmptar} --exclude=target --exclude=node_modules --exclude=.git --exclude=.mise/logs --exclude=.mise/state --exclude=.build --exclude=.gradle -C ${PROJECT_DIR} .`;
  await scp(profile, tmptar, `${profile.user}@127.0.0.1:sync.tar.gz`);
  await ssh(
    profile,
    `if not exist "${vmProjectDir}" mkdir "${vmProjectDir}" && cd "${vmProjectDir}" && tar -xzf "%USERPROFILE%\\sync.tar.gz" && del "%USERPROFILE%\\sync.tar.gz"`,
  );
  ok("Code synced", LOG);
} finally {
  await $`rm -f ${tmptar}`.nothrow();
}

// ── Build inside VM ─────────────────────────────────────────────────────

info("Installing tools inside VM (mise install)...", LOG);
const install = await ssh(profile, `cd "${vmProjectDir}" && mise trust && mise install`);
if (install.exitCode !== 0) die("mise install failed inside VM");
ok("Tools installed", LOG);

info("Building Tauri Windows app (this takes a while on first run)...", LOG);
const build = await ssh(profile, `cd "${vmProjectDir}" && mise run build`);
if (build.exitCode !== 0) die("Build failed inside VM");
ok("Build complete", LOG);

// ── Pull artifacts back ─────────────────────────────────────────────────

info("Pulling artifacts...", LOG);
mkdirSync(artifactsDir, { recursive: true });

await ssh(
  profile,
  `cd "${vmProjectDir}\\src-tauri\\target\\release\\bundle" && tar -czf "%USERPROFILE%\\artifacts.tar.gz" .`,
);
await scp(profile, `${profile.user}@127.0.0.1:artifacts.tar.gz`, join(artifactsDir, "artifacts.tar.gz"));
await $`tar -xzf ${join(artifactsDir, "artifacts.tar.gz")} -C ${artifactsDir}`;
await $`rm -f ${join(artifactsDir, "artifacts.tar.gz")}`.nothrow();
await ssh(profile, `del "%USERPROFILE%\\artifacts.tar.gz"`).catch(() => {});

log("", LOG);
ok(`Windows build artifacts in ${artifactsDir}:`, LOG);

const glob = new Bun.Glob("**/*.{msi,exe}");
for await (const path of glob.scan(artifactsDir)) {
  const file = Bun.file(join(artifactsDir, path));
  const size = (file.size / 1024 / 1024).toFixed(1);
  log(`  ${path} (${size} MB)`, LOG);
}
