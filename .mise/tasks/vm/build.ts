#!/usr/bin/env bun

//MISE description="Build app in the VM"
//MISE alias="vm-build"
//MISE hide=true

import { $ } from "bun";
import { mkdirSync } from "fs";
import { join } from "path";
import {
  PROJECT_DIR, PROJECT_NAME,
  parseVMArg, getProfile, vmHome,
  ensureSshpass, ssh, scp, info, ok, die, log, timestamp,
} from "../_lib.ts";

const LOG = "vm-build.log";
log(`── ${timestamp()} ──`, LOG);

// Uses "windows-build" for Windows, "linux-build" for Linux — or specify explicitly
const { vmName } = parseVMArg();
const profile = getProfile(vmName);
const home = vmHome(profile);
const sep = profile.os === "windows" ? "\\" : "/";
const vmProjectDir = `${home}${sep}${PROJECT_NAME}`;
const platformDir = profile.os === "windows" ? "windows" : "linux";
const artifactsDir = join(PROJECT_DIR, ".build", platformDir);

await ensureSshpass();

// Auto-start VM if not reachable
const probe = await ssh(profile, "echo ok", { quiet: true }).catch(() => ({ stdout: "", exitCode: 1 }));
if (!probe.stdout.includes("ok")) {
  info(`${vmName} VM not reachable — starting it...`, LOG);
  const upScript = join(import.meta.dir, "up.ts");
  await $`bun ${upScript} ${vmName}`;
}

// ── Sync code to VM ─────────────────────────────────────────────────────

info(`Syncing ${PROJECT_NAME} to build VM...`, LOG);
const tmptar = `/tmp/vm-build-${Date.now()}.tar.gz`;
try {
  await $`tar -czf ${tmptar} --exclude=target --exclude=node_modules --exclude=.git --exclude=.mise/logs --exclude=.mise/state --exclude=.build --exclude=.gradle -C ${PROJECT_DIR} .`;
  await scp(profile, tmptar, `${profile.user}@127.0.0.1:sync.tar.gz`);
  if (profile.os === "linux") {
    await ssh(
      profile,
      `mkdir -p "${vmProjectDir}" && cd "${vmProjectDir}" && tar -xzf ~/sync.tar.gz && rm ~/sync.tar.gz`,
    );
  } else {
    await ssh(
      profile,
      `if not exist "${vmProjectDir}" mkdir "${vmProjectDir}" && cd "${vmProjectDir}" && tar -xzf "%USERPROFILE%\\sync.tar.gz" && del "%USERPROFILE%\\sync.tar.gz"`,
    );
  }
  ok("Code synced", LOG);
} finally {
  await $`rm -f ${tmptar}`.nothrow();
}

// ── Build inside VM ─────────────────────────────────────────────────────

info("Installing tools inside VM (mise install)...", LOG);
const install = await ssh(profile, `cd "${vmProjectDir}" && mise trust && mise install`);
if (install.exitCode !== 0) die("mise install failed inside VM");
ok("Tools installed", LOG);

const platformLabel = profile.os === "windows" ? "Windows" : "Linux";
info(`Building Tauri ${platformLabel} app (this takes a while on first run)...`, LOG);
const build = await ssh(profile, `cd "${vmProjectDir}" && mise run build`);
if (build.exitCode !== 0) die("Build failed inside VM");
ok("Build complete", LOG);

// ── Pull artifacts back ─────────────────────────────────────────────────

info("Pulling artifacts...", LOG);
mkdirSync(artifactsDir, { recursive: true });

const bundlePath = `${vmProjectDir}${sep}src-tauri${sep}target${sep}release${sep}bundle`;
if (profile.os === "linux") {
  await ssh(profile, `cd "${bundlePath}" && tar -czf ~/artifacts.tar.gz .`);
  await scp(profile, `${profile.user}@127.0.0.1:artifacts.tar.gz`, join(artifactsDir, "artifacts.tar.gz"));
  await ssh(profile, `rm ~/artifacts.tar.gz`).catch(() => {});
} else {
  await ssh(profile, `cd "${bundlePath}" && tar -czf "%USERPROFILE%\\artifacts.tar.gz" .`);
  await scp(profile, `${profile.user}@127.0.0.1:artifacts.tar.gz`, join(artifactsDir, "artifacts.tar.gz"));
  await ssh(profile, `del "%USERPROFILE%\\artifacts.tar.gz"`).catch(() => {});
}
await $`tar -xzf ${join(artifactsDir, "artifacts.tar.gz")} -C ${artifactsDir}`;
await $`rm -f ${join(artifactsDir, "artifacts.tar.gz")}`.nothrow();

log("", LOG);
ok(`Build artifacts in ${artifactsDir}:`, LOG);

const artifactGlob = profile.os === "linux" ? "**/*.{deb,AppImage,rpm}" : "**/*.{msi,exe}";
const glob = new Bun.Glob(artifactGlob);
for await (const path of glob.scan(artifactsDir)) {
  const file = Bun.file(join(artifactsDir, path));
  const size = (file.size / 1024 / 1024).toFixed(1);
  log(`  ${path} (${size} MB)`, LOG);
}
