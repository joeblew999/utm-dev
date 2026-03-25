#!/usr/bin/env bun

//MISE description="Check what's installed and what's missing"
//MISE alias="d"

import { $ } from "bun";
import { existsSync } from "fs";
import { UTMCTL, VM_PROFILES, hasState, loadState, log } from "./_lib.ts";

async function check(name: string, testCmd: string, verCmd: string): Promise<void> {
  const test = await $`sh -c ${testCmd}`.quiet().nothrow();
  if (test.exitCode === 0) {
    const ver = await $`sh -c ${verCmd}`.quiet().nothrow();
    const v = ver.stdout.toString().trim().split("\n")[0];
    log(`  ✓ ${name}${v ? ` (${v})` : ""}`);
  } else {
    log(`  ✗ ${name}`);
  }
}

const IS_LINUX = process.platform === "linux";
const IS_MACOS = process.platform === "darwin";

log("═══ utm-dev doctor ═══");
log(`  Platform: ${IS_LINUX ? "Linux" : IS_MACOS ? "macOS" : process.platform}`);
log("");

// ── Common tools (all platforms) ──
log("── Dev tools ──");
await check("mise", "command -v mise", "mise --version");
await check("Rust", "command -v cargo", "cargo --version | awk '{print $2}'");
await check("cargo-tauri", "cargo tauri --version", "cargo tauri --version | awk '{print $2}'");
await check("bun", "command -v bun", "bun --version");
log("");

if (IS_LINUX) {
  // ── Linux system deps ──
  log("── Tauri system deps (apt) ──");
  const linuxDeps = [
    "build-essential", "pkg-config", "libwebkit2gtk-4.1-dev", "libgtk-3-dev",
    "libjavascriptcoregtk-4.1-dev", "libsoup-3.0-dev",
    "libayatana-appindicator3-dev", "librsvg2-dev", "libssl-dev", "libxdo-dev",
    "patchelf",
  ];
  for (const pkg of linuxDeps) {
    await check(pkg, `dpkg -s ${pkg} 2>/dev/null | grep -q "ok installed"`, "echo installed");
  }
  log("");
}

if (IS_MACOS) {
  // ── macOS-specific tools ──
  log("── macOS tools ──");
  await check("Xcode", "xcode-select -p", "xcode-select -p");
  await check("Homebrew", "command -v brew", "brew --version | head -1");
  await check("sshpass", "command -v sshpass", "echo installed");
  log("");

  log("── iOS ──");
  await check("CocoaPods", "command -v pod", "pod --version");
  await check("xcodegen", "command -v xcodegen", "xcodegen --version 2>&1 | head -1");
  await check(
    "Simulator",
    "xcrun simctl list devices booted 2>/dev/null | grep -q Booted",
    "xcrun simctl list devices booted 2>/dev/null | grep Booted | head -1 | sed 's/.*(//' | sed 's/).*//'"
  );
  log("");

  log("── Android ──");
  const androidHome = process.env.ANDROID_HOME ?? "/nonexistent";
  const ndkHome = process.env.NDK_HOME ?? "/nonexistent";
  await check("ANDROID_HOME", `test -d "${androidHome}"`, `echo ${androidHome}`);
  await check("NDK", `test -f "${ndkHome}/source.properties"`, `echo ${ndkHome}`);
  await check("Java", "command -v java", "java -version 2>&1 | head -1");
  await check(
    "sdkmanager",
    `test -x "${androidHome}/cmdline-tools/latest/bin/sdkmanager"`,
    "echo installed"
  );
  await check("emulator", `test -x "${androidHome}/emulator/emulator"`, "echo installed");
  await check(
    "system-image",
    `test -d "${androidHome}/system-images/android-35/google_apis/arm64-v8a"`,
    "echo android-35 arm64-v8a"
  );
  await check(
    "AVD (utm-dev)",
    `test -d "${process.env.HOME}/.android/avd/utm-dev.avd"`,
    "echo created"
  );
  log("");
}

// VMs and cache are macOS-only (Linux doesn't run UTM)
if (IS_MACOS) {
  log("── VMs ──");
  await check("UTM", `test -x ${UTMCTL}`, `${UTMCTL} version 2>/dev/null || echo installed`);

  const sshpassAvail = (await $`command -v sshpass`.quiet().nothrow()).exitCode === 0;

  for (const [vmName, profile] of Object.entries(VM_PROFILES)) {
    if (!hasState(vmName)) {
      log(`  ✗ ${vmName} VM (not created)`);
      continue;
    }

    const { VM_UUID, VM_DISPLAY_NAME } = loadState(vmName);
    const list = await $`${UTMCTL} list`.quiet().nothrow();
    const vmLine = list.stdout.toString().split("\n").find((l) => l.includes(VM_UUID));

    if (!vmLine) {
      log(`  ✗ ${vmName} VM (state exists but VM not in UTM)`);
      continue;
    }

    const status = vmLine.split(/\s+/)[1];
    log(`  ✓ ${vmName} VM (${status}) — ${VM_DISPLAY_NAME}`);

    // Check SSH
    if (sshpassAvail && status === "started") {
      const sshTest = await $`sshpass -p ${profile.pass} ssh -o ConnectTimeout=2 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p ${profile.sshPort} ${profile.user}@127.0.0.1 "echo ok"`
        .quiet().nothrow();
      if (sshTest.stdout.toString().includes("ok")) {
        log(`    SSH: port ${profile.sshPort}`);
      } else {
        log(`    SSH: not reachable (port ${profile.sshPort})`);
      }
    }

    // Check WinRM (Windows only)
    if (status === "started" && profile.os === "windows" && profile.winrmPort) {
      try {
        await fetch(`http://127.0.0.1:${profile.winrmPort}/wsman`, { signal: AbortSignal.timeout(2000) });
        log(`    WinRM: port ${profile.winrmPort}`);
      } catch {
        log(`    WinRM: not reachable (port ${profile.winrmPort})`);
      }
    }
  }

  log("");

  // Box cache
  const cacheDir = `${process.env.HOME}/.cache/utm-dev`;
  if (existsSync(cacheDir)) {
    const du = await $`du -sh ${cacheDir}`.quiet().nothrow();
    const size = du.stdout.toString().split("\t")[0];
    log("── Cache ──");
    log(`  ✓ Box cache (${size} at ~/.cache/utm-dev/)`);
  } else {
    log("── Cache ──");
    log("  ✗ No box cache (first vm:up will download ~6 GB)");
  }

  log("");
}

log("");
