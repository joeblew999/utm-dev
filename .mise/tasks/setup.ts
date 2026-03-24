#!/usr/bin/env bun

// #MISE description="Install all prerequisites for Tauri development"
// #MISE alias="s"

// Stages:
//   1. Host tools   — Rust, Xcode check
//   2. Mobile SDKs  — Android SDK, NDK, platform-tools, build-tools, JDK
//   3. Rust targets — Android cross-compilation targets
//   4. iOS deps     — CocoaPods
//
// cargo-tauri and bun are managed by mise [tools] — not installed here.
// sshpass is auto-installed by vm:* tasks on first use.
// Every step is idempotent. Run it 100 times, nothing breaks.

import { $ } from "bun";
import { existsSync, mkdirSync, appendFileSync } from "fs";
import { join } from "path";

const PROJECT_DIR = process.cwd();
const LOGDIR = join(PROJECT_DIR, ".mise", "logs");
mkdirSync(LOGDIR, { recursive: true });

const LOG = join(LOGDIR, "setup.log");

function log(msg: string) {
  const line = msg + "\n";
  process.stdout.write(line);
  appendFileSync(LOG, line);
}

const info = (msg: string) => log(`→ ${msg}`);
const ok = (msg: string) => log(`✓ ${msg}`);

function die(msg: string): never {
  log(`✗ ${msg}`);
  process.exit(1);
}

async function cmdExists(cmd: string): Promise<boolean> {
  return (await $`command -v ${cmd}`.quiet().nothrow()).exitCode === 0;
}

async function dirExists(path: string): Promise<boolean> {
  return existsSync(path);
}

log(`── ${new Date().toISOString().replace("T", " ").slice(0, 19)} ──`);

const JAVA_VERSION = "temurin-17.0.18+8";
const NDK_VERSION = "27.2.12479018";
const BUILD_TOOLS_VERSION = "35.0.0";
const PLATFORM_VERSION = "android-35";
const CMDLINE_TOOLS_URL =
  "https://dl.google.com/android/repository/commandlinetools-mac-14742923_latest.zip";
const ANDROID_HOME = process.env.ANDROID_HOME ?? `${process.env.HOME}/.android-sdk`;

log("═══ utm-dev setup ═══");
log("");

// ── Stage 1: Host tools ───────────────────────────────────────────────────

log("── Stage 1: Host tools ──");

if (await cmdExists("cargo")) {
  const ver = (await $`cargo --version`.quiet()).stdout.toString().trim().split(" ")[1];
  ok(`Rust ${ver}`);
} else {
  info("Installing Rust...");
  await $`curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y`;
  ok("Rust installed");
}

if ((await $`xcode-select -p`.quiet().nothrow()).exitCode === 0) {
  const path = (await $`xcode-select -p`.quiet()).stdout.toString().trim();
  ok(`Xcode (${path})`);
} else {
  die(
    "Xcode not found.\n  Install from: https://apps.apple.com/app/xcode/id497799835\n  Then run: sudo xcode-select --switch /Applications/Xcode.app",
  );
}

log("");

// ── Stage 2: Mobile SDKs ──────────────────────────────────────────────────

log("── Stage 2: Mobile SDKs ──");

// Java (via mise)
if ((await $`mise where java`.quiet().nothrow()).exitCode === 0) {
  const javaPath = (await $`mise where java`.quiet()).stdout.toString().trim();
  ok(`Java (${javaPath})`);
} else {
  info(`Installing Java ${JAVA_VERSION} via mise...`);
  await $`mise use --global java@${JAVA_VERSION}`;
  ok("Java installed");
}

// Android SDK cmdline-tools
const sdkmanager = `${ANDROID_HOME}/cmdline-tools/latest/bin/sdkmanager`;
if (existsSync(sdkmanager)) {
  ok(`Android cmdline-tools (${ANDROID_HOME})`);
} else {
  info(`Installing Android cmdline-tools to ${ANDROID_HOME}...`);
  mkdirSync(ANDROID_HOME, { recursive: true });
  const tmpzip = `/tmp/cmdline-tools-${Date.now()}.zip`;
  await $`curl -sSfL -o ${tmpzip} ${CMDLINE_TOOLS_URL}`;
  await $`unzip -qo ${tmpzip} -d ${ANDROID_HOME}/cmdline-tools-tmp`;
  mkdirSync(`${ANDROID_HOME}/cmdline-tools`, { recursive: true });
  await $`rm -rf ${ANDROID_HOME}/cmdline-tools/latest`.nothrow();
  await $`mv ${ANDROID_HOME}/cmdline-tools-tmp/cmdline-tools ${ANDROID_HOME}/cmdline-tools/latest`;
  await $`rmdir ${ANDROID_HOME}/cmdline-tools-tmp`.nothrow();
  await $`rm -f ${tmpzip}`;
  ok("Android cmdline-tools installed");
}

// Set up paths for sdkmanager commands
const JAVA_HOME = (await $`mise where java`.quiet()).stdout.toString().trim();
const sdkEnv = {
  ...process.env,
  ANDROID_HOME,
  JAVA_HOME,
  PATH: `${ANDROID_HOME}/cmdline-tools/latest/bin:${ANDROID_HOME}/platform-tools:${process.env.PATH}`,
};

// Accept licenses
info("Accepting Android SDK licenses...");
await $`yes 2>/dev/null | ${sdkmanager} --licenses --sdk_root=${ANDROID_HOME} > /dev/null 2>&1 || true`
  .env(sdkEnv)
  .nothrow();

// Platform
if (existsSync(`${ANDROID_HOME}/platforms/${PLATFORM_VERSION}`)) {
  ok(`Android platform ${PLATFORM_VERSION}`);
} else {
  info(`Installing Android platform ${PLATFORM_VERSION}...`);
  await $`${sdkmanager} --sdk_root=${ANDROID_HOME} platforms;${PLATFORM_VERSION}`.env(sdkEnv);
  ok("Android platform installed");
}

// Build tools
if (existsSync(`${ANDROID_HOME}/build-tools/${BUILD_TOOLS_VERSION}`)) {
  ok(`Android build-tools ${BUILD_TOOLS_VERSION}`);
} else {
  info(`Installing Android build-tools ${BUILD_TOOLS_VERSION}...`);
  await $`${sdkmanager} --sdk_root=${ANDROID_HOME} build-tools;${BUILD_TOOLS_VERSION}`.env(sdkEnv);
  ok("Android build-tools installed");
}

// Platform tools (adb)
if (existsSync(`${ANDROID_HOME}/platform-tools`)) {
  ok("Android platform-tools (adb)");
} else {
  info("Installing Android platform-tools...");
  await $`${sdkmanager} --sdk_root=${ANDROID_HOME} platform-tools`.env(sdkEnv);
  ok("Android platform-tools installed");
}

// NDK
if (existsSync(`${ANDROID_HOME}/ndk/${NDK_VERSION}/source.properties`)) {
  ok(`Android NDK ${NDK_VERSION}`);
} else {
  await $`rm -rf ${ANDROID_HOME}/ndk/${NDK_VERSION}`.nothrow();
  info(`Installing Android NDK ${NDK_VERSION}...`);
  await $`${sdkmanager} --sdk_root=${ANDROID_HOME} ndk;${NDK_VERSION}`.env(sdkEnv);
  ok("Android NDK installed");
}

log("");

// ── Stage 3: Rust targets ─────────────────────────────────────────────────

log("── Stage 3: Rust Android targets ──");

const TARGETS = [
  "aarch64-linux-android",
  "armv7-linux-androideabi",
  "i686-linux-android",
  "x86_64-linux-android",
];

for (const target of TARGETS) {
  const installed = await $`rustup target list --installed`.quiet();
  if (installed.stdout.toString().includes(target)) {
    ok(target);
  } else {
    info(`Adding ${target}...`);
    await $`rustup target add ${target}`;
    ok(`${target} added`);
  }
}

log("");

// ── Stage 4: iOS deps ─────────────────────────────────────────────────────

log("── Stage 4: iOS deps ──");

if (await cmdExists("pod")) {
  const ver = (await $`pod --version`.quiet()).stdout.toString().trim();
  ok(`CocoaPods ${ver}`);
} else {
  info("Installing CocoaPods...");
  await $`gem install cocoapods`;
  ok("CocoaPods installed");
}

log("");

// ── Summary ───────────────────────────────────────────────────────────────

log("═══ Setup complete ═══");
log("");
log("Environment variables (set in mise.toml):");
log(`  ANDROID_HOME=${ANDROID_HOME}`);
log(`  NDK_HOME=${ANDROID_HOME}/ndk/${NDK_VERSION}`);
log(`  JAVA_HOME=${JAVA_HOME}`);
