#!/usr/bin/env bun

//MISE description="Install Mac + mobile dev tools (Rust, Android SDK, iOS)"
//MISE alias="s"

// Stages:
//   1. Host tools   — Rust, Xcode check
//   2. Mobile SDKs  — Android SDK, NDK, platform-tools, build-tools, emulator, JDK
//   3. Rust targets — Android cross-compilation targets
//   4. iOS deps     — CocoaPods
//
// Windows VM setup is handled lazily by vm:up on first run.
// cargo-tauri and bun are managed by mise [tools] — not installed here.
// Every step is idempotent. Run it 100 times, nothing breaks.

import { $ } from "bun";
import { existsSync, mkdirSync } from "fs";
import { info, ok, die, log, timestamp } from "./_lib.ts";

const LOG = "setup.log";

async function cmdExists(cmd: string): Promise<boolean> {
  return (await $`command -v ${cmd}`.quiet().nothrow()).exitCode === 0;
}

log(`── ${timestamp()} ──`, LOG);

const JAVA_VERSION = "temurin-17.0.18+8";
const NDK_VERSION = "27.2.12479018";
const BUILD_TOOLS_VERSION = "35.0.0";
const PLATFORM_VERSION = "android-35";
const CMDLINE_TOOLS_URL =
  "https://dl.google.com/android/repository/commandlinetools-mac-14742923_latest.zip";
const SYSTEM_IMAGE = `system-images;${PLATFORM_VERSION};google_apis;arm64-v8a`;
const AVD_NAME = "utm-dev";
const ANDROID_HOME = process.env.ANDROID_HOME ?? `${process.env.HOME}/.android-sdk`;

log("═══ utm-dev setup ═══", LOG);
log("", LOG);

// ── Stage 1: Host tools ───────────────────────────────────────────────────

log("── Stage 1: Host tools ──", LOG);

if (await cmdExists("cargo")) {
  const ver = (await $`cargo --version`.quiet()).stdout.toString().trim().split(" ")[1];
  ok(`Rust ${ver}`, LOG);
} else {
  info("Installing Rust...", LOG);
  await $`curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y`;
  ok("Rust installed", LOG);
}

if ((await $`xcode-select -p`.quiet().nothrow()).exitCode === 0) {
  const path = (await $`xcode-select -p`.quiet()).stdout.toString().trim();
  ok(`Xcode (${path})`, LOG);
} else {
  die(
    "Xcode not found.\n  Install from: https://apps.apple.com/app/xcode/id497799835\n  Then run: sudo xcode-select --switch /Applications/Xcode.app",
  );
}

log("", LOG);

// ── Stage 2: Mobile SDKs ──────────────────────────────────────────────────

log("── Stage 2: Mobile SDKs ──", LOG);

// Java (via mise)
if ((await $`mise where java`.quiet().nothrow()).exitCode === 0) {
  const javaPath = (await $`mise where java`.quiet()).stdout.toString().trim();
  ok(`Java (${javaPath})`, LOG);
} else {
  info(`Installing Java ${JAVA_VERSION} via mise...`, LOG);
  await $`mise use --global java@${JAVA_VERSION}`;
  ok("Java installed", LOG);
}

// Android SDK cmdline-tools
const sdkmanager = `${ANDROID_HOME}/cmdline-tools/latest/bin/sdkmanager`;
if (existsSync(sdkmanager)) {
  ok(`Android cmdline-tools (${ANDROID_HOME})`, LOG);
} else {
  info(`Installing Android cmdline-tools to ${ANDROID_HOME}...`, LOG);
  mkdirSync(ANDROID_HOME, { recursive: true });
  const tmpzip = `/tmp/cmdline-tools-${Date.now()}.zip`;
  await $`curl -sSfL -o ${tmpzip} ${CMDLINE_TOOLS_URL}`;
  await $`unzip -qo ${tmpzip} -d ${ANDROID_HOME}/cmdline-tools-tmp`;
  mkdirSync(`${ANDROID_HOME}/cmdline-tools`, { recursive: true });
  await $`rm -rf ${ANDROID_HOME}/cmdline-tools/latest`.nothrow();
  await $`mv ${ANDROID_HOME}/cmdline-tools-tmp/cmdline-tools ${ANDROID_HOME}/cmdline-tools/latest`;
  await $`rmdir ${ANDROID_HOME}/cmdline-tools-tmp`.nothrow();
  await $`rm -f ${tmpzip}`;
  ok("Android cmdline-tools installed", LOG);
}

// Set up paths for sdkmanager commands
const JAVA_HOME = (await $`mise where java`.quiet()).stdout.toString().trim();
const sdkEnv = {
  ...process.env,
  ANDROID_HOME,
  JAVA_HOME,
  PATH: `${ANDROID_HOME}/cmdline-tools/latest/bin:${ANDROID_HOME}/platform-tools:${ANDROID_HOME}/emulator:${process.env.PATH}`,
};

// Accept licenses
info("Accepting Android SDK licenses...", LOG);
await $`sh -c ${"yes 2>/dev/null | " + sdkmanager + " --licenses --sdk_root=" + ANDROID_HOME + " >/dev/null 2>&1 || true"}`
  .env(sdkEnv)
  .quiet()
  .nothrow();

// Platform
if (existsSync(`${ANDROID_HOME}/platforms/${PLATFORM_VERSION}`)) {
  ok(`Android platform ${PLATFORM_VERSION}`, LOG);
} else {
  info(`Installing Android platform ${PLATFORM_VERSION}...`, LOG);
  await $`${sdkmanager} --sdk_root=${ANDROID_HOME} platforms;${PLATFORM_VERSION}`.env(sdkEnv);
  ok("Android platform installed", LOG);
}

// Build tools
if (existsSync(`${ANDROID_HOME}/build-tools/${BUILD_TOOLS_VERSION}`)) {
  ok(`Android build-tools ${BUILD_TOOLS_VERSION}`, LOG);
} else {
  info(`Installing Android build-tools ${BUILD_TOOLS_VERSION}...`, LOG);
  await $`${sdkmanager} --sdk_root=${ANDROID_HOME} build-tools;${BUILD_TOOLS_VERSION}`.env(sdkEnv);
  ok("Android build-tools installed", LOG);
}

// Platform tools (adb)
if (existsSync(`${ANDROID_HOME}/platform-tools`)) {
  ok("Android platform-tools (adb)", LOG);
} else {
  info("Installing Android platform-tools...", LOG);
  await $`${sdkmanager} --sdk_root=${ANDROID_HOME} platform-tools`.env(sdkEnv);
  ok("Android platform-tools installed", LOG);
}

// NDK
if (existsSync(`${ANDROID_HOME}/ndk/${NDK_VERSION}/source.properties`)) {
  ok(`Android NDK ${NDK_VERSION}`, LOG);
} else {
  await $`rm -rf ${ANDROID_HOME}/ndk/${NDK_VERSION}`.nothrow();
  info(`Installing Android NDK ${NDK_VERSION}...`, LOG);
  await $`${sdkmanager} --sdk_root=${ANDROID_HOME} ndk;${NDK_VERSION}`.env(sdkEnv);
  ok("Android NDK installed", LOG);
}

// Emulator
if (existsSync(`${ANDROID_HOME}/emulator/emulator`)) {
  ok("Android emulator", LOG);
} else {
  info("Installing Android emulator...", LOG);
  await $`${sdkmanager} --sdk_root=${ANDROID_HOME} emulator`.env(sdkEnv);
  ok("Android emulator installed", LOG);
}

// System image (ARM64 for Apple Silicon)
const imageDir = `${ANDROID_HOME}/system-images/${PLATFORM_VERSION}/google_apis/arm64-v8a`;
if (existsSync(imageDir)) {
  ok(`System image ${SYSTEM_IMAGE}`, LOG);
} else {
  info(`Installing system image (ARM64)... this takes a while`, LOG);
  await $`${sdkmanager} --sdk_root=${ANDROID_HOME} ${SYSTEM_IMAGE}`.env(sdkEnv);
  ok("System image installed", LOG);
}

// AVD
const avdmanager = `${ANDROID_HOME}/cmdline-tools/latest/bin/avdmanager`;
const avdList = (await $`${avdmanager} list avd -c`.env(sdkEnv).quiet().nothrow()).stdout.toString();
if (avdList.includes(AVD_NAME)) {
  ok(`AVD "${AVD_NAME}"`, LOG);
} else {
  info(`Creating AVD "${AVD_NAME}"...`, LOG);
  await $`${avdmanager} create avd -n ${AVD_NAME} -k ${SYSTEM_IMAGE} --device pixel_6 --force`.env(sdkEnv);
  ok(`AVD "${AVD_NAME}" created`, LOG);
}

log("", LOG);

// ── Stage 3: Rust targets ─────────────────────────────────────────────────

log("── Stage 3: Rust Android targets ──", LOG);

const TARGETS = [
  "aarch64-linux-android",
  "armv7-linux-androideabi",
  "i686-linux-android",
  "x86_64-linux-android",
];

for (const target of TARGETS) {
  const installed = await $`rustup target list --installed`.quiet();
  if (installed.stdout.toString().includes(target)) {
    ok(target, LOG);
  } else {
    info(`Adding ${target}...`, LOG);
    await $`rustup target add ${target}`;
    ok(`${target} added`, LOG);
  }
}

log("", LOG);

// ── Stage 4: iOS deps ─────────────────────────────────────────────────────

log("── Stage 4: iOS deps ──", LOG);

if (await cmdExists("pod")) {
  const ver = (await $`pod --version`.quiet()).stdout.toString().trim();
  ok(`CocoaPods ${ver}`, LOG);
} else {
  info("Installing CocoaPods...", LOG);
  await $`gem install cocoapods`;
  ok("CocoaPods installed", LOG);
}

// ── Done ─────────────────────────────────────────────────────────────────

log("", LOG);
log("═══ Setup complete ═══", LOG);
log("", LOG);
log("Environment:", LOG);
log(`  ANDROID_HOME=${ANDROID_HOME}`, LOG);
log(`  NDK_HOME=${ANDROID_HOME}/ndk/${NDK_VERSION}`, LOG);
log(`  JAVA_HOME=${JAVA_HOME}`, LOG);
log("", LOG);
log("Next:", LOG);
log("  mise run mac:dev            # macOS desktop dev mode", LOG);
log("  mise run ios:sim            # iOS simulator", LOG);
log("  mise run android:sim        # Android emulator", LOG);
log("  mise run windows:build      # Windows .msi/.exe (VM auto-starts)", LOG);
log("  mise run linux:build        # Linux .deb/.AppImage (VM auto-starts)", LOG);
log("  mise run doctor             # check everything", LOG);
