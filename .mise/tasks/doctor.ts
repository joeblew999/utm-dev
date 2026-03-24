#!/usr/bin/env bun

//MISE description="Check what's installed and what's missing"
//MISE alias="d"

import { $ } from "bun";
import { existsSync } from "fs";
import { UTMCTL, VM_PROFILES, hasState, loadState } from "./_lib.ts";

async function check(name: string, testCmd: string, verCmd: string): Promise<void> {
  const test = await $`sh -c ${testCmd}`.quiet().nothrow();
  if (test.exitCode === 0) {
    const ver = await $`sh -c ${verCmd}`.quiet().nothrow();
    const v = ver.stdout.toString().trim().split("\n")[0];
    console.log(`  ✓ ${name}${v ? ` (${v})` : ""}`);
  } else {
    console.log(`  ✗ ${name}`);
  }
}

console.log("═══ utm-dev doctor ═══");
console.log("");

console.log("── Mac tools ──");
await check("mise", "command -v mise", "mise --version");
await check("Rust", "command -v cargo", "cargo --version | awk '{print $2}'");
await check("cargo-tauri", "cargo tauri --version", "cargo tauri --version | awk '{print $2}'");
await check("Xcode", "xcode-select -p", "xcode-select -p");
await check("Homebrew", "command -v brew", "brew --version | head -1");
await check("sshpass", "command -v sshpass", "echo installed");
await check("bun", "command -v bun", "bun --version");
console.log("");

console.log("── iOS ──");
await check("CocoaPods", "command -v pod", "pod --version");
await check("xcodegen", "command -v xcodegen", "xcodegen --version 2>&1 | head -1");
await check(
  "Simulator",
  "xcrun simctl list devices booted 2>/dev/null | grep -q Booted",
  "xcrun simctl list devices booted 2>/dev/null | grep Booted | head -1 | sed 's/.*(//' | sed 's/).*//'"
);
console.log("");

console.log("── Android ──");
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
console.log("");

console.log("── Windows VMs ──");
await check("UTM", `test -x ${UTMCTL}`, `${UTMCTL} version 2>/dev/null || echo installed`);

const sshpassAvail = (await $`command -v sshpass`.quiet().nothrow()).exitCode === 0;

for (const [vmName, profile] of Object.entries(VM_PROFILES)) {
  if (!hasState(vmName)) {
    console.log(`  ✗ ${vmName} VM (not created)`);
    continue;
  }

  const { VM_UUID, VM_DISPLAY_NAME } = loadState(vmName);
  const list = await $`${UTMCTL} list`.quiet().nothrow();
  const vmLine = list.stdout.toString().split("\n").find((l) => l.includes(VM_UUID));

  if (!vmLine) {
    console.log(`  ✗ ${vmName} VM (state exists but VM not in UTM)`);
    continue;
  }

  const status = vmLine.split(/\s+/)[1];
  console.log(`  ✓ ${vmName} VM (${status}) — ${VM_DISPLAY_NAME}`);

  // Check SSH
  if (sshpassAvail && status === "started") {
    const sshTest = await $`sshpass -p ${profile.pass} ssh -o ConnectTimeout=2 -o StrictHostKeyChecking=no -p ${profile.sshPort} ${profile.user}@127.0.0.1 "echo ok"`
      .quiet().nothrow();
    if (sshTest.stdout.toString().includes("ok")) {
      console.log(`    SSH: port ${profile.sshPort}`);
    } else {
      console.log(`    SSH: not reachable (port ${profile.sshPort})`);
    }
  }

  // Check WinRM
  if (status === "started") {
    try {
      await fetch(`http://127.0.0.1:${profile.winrmPort}/wsman`, { signal: AbortSignal.timeout(2000) });
      console.log(`    WinRM: port ${profile.winrmPort}`);
    } catch {
      console.log(`    WinRM: not reachable (port ${profile.winrmPort})`);
    }
  }
}

console.log("");

// Box cache
const cacheDir = `${process.env.HOME}/.cache/utm-dev`;
if (existsSync(cacheDir)) {
  const du = await $`du -sh ${cacheDir}`.quiet().nothrow();
  const size = du.stdout.toString().split("\t")[0];
  console.log("── Cache ──");
  console.log(`  ✓ Box cache (${size} at ~/.cache/utm-dev/)`);
} else {
  console.log("── Cache ──");
  console.log("  ✗ No box cache (first vm:up will download ~6 GB)");
}

console.log("");
