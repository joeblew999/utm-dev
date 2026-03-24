#!/usr/bin/env bun

// #MISE description="Check what's installed and what's missing"
// #MISE alias="d"

// Quick health check — shows what's working and what needs fixing.

import { $ } from "bun";
import { existsSync } from "fs";
import { SSH_PORT, VM_USER, VM_PASS, UTMCTL } from "./_lib.ts";

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

console.log("── Windows VM ──");
await check(
  "UTM",
  `test -x ${UTMCTL}`,
  `${UTMCTL} version 2>/dev/null || echo installed`
);

// Check VM status
const utmExists = await $`test -x ${UTMCTL}`.quiet().nothrow();
if (utmExists.exitCode === 0) {
  const list = await $`${UTMCTL} list`.quiet().nothrow();
  const vmLine = (list.stdout?.toString() ?? "")
    .split("\n")
    .filter((l) => l.trim() && !l.startsWith("UUID"))
    .find((l) => l.trim());
  if (vmLine) {
    const status = vmLine.trim().split(/\s+/)[1];
    console.log(`  ✓ VM (${status})`);
  } else {
    console.log("  ✗ VM (not imported)");
  }
} else {
  console.log("  ✗ VM (UTM not installed)");
}

// Check SSH
const sshpassCheck = await $`command -v sshpass`.quiet().nothrow();
if (sshpassCheck.exitCode === 0) {
  const sshTest = await $`sshpass -p ${VM_PASS} ssh -o ConnectTimeout=2 -o StrictHostKeyChecking=no -p ${SSH_PORT} ${VM_USER}@127.0.0.1 "echo ok"`
    .quiet()
    .nothrow();
  if (sshTest.stdout.toString().includes("ok")) {
    console.log(`  ✓ SSH (port ${SSH_PORT})`);
  } else {
    console.log("  ✗ SSH (not reachable — VM stopped or not bootstrapped)");
  }
} else {
  console.log("  ✗ SSH (sshpass not installed)");
}

// Check WinRM
try {
  await fetch(`http://127.0.0.1:5985/wsman`, { signal: AbortSignal.timeout(2000) });
  console.log("  ✓ WinRM (port 5985)");
} catch {
  console.log("  ✗ WinRM (not reachable)");
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
