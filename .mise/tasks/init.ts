#!/usr/bin/env bun

//MISE description="Add utm-dev tools and env to your project's mise.toml"
//MISE alias="i"

// Adds the [tools] and [env] blocks needed for Tauri cross-platform builds.
// Idempotent — safe to run multiple times.

import { existsSync, readFileSync, appendFileSync } from "fs";
import { join } from "path";
import { log, die } from "./_lib.ts";

const miseToml = join(process.cwd(), "mise.toml");

if (!existsSync(miseToml)) {
  die(`No mise.toml found in ${process.cwd()} — create one first, or run from your project root.`);
}

const content = readFileSync(miseToml, "utf-8");

// Check if we already added our blocks
if (content.includes("# utm-dev tools")) {
  log("✓ Already initialised");
  process.exit(0);
}

const hasTools = /^\[tools\]/m.test(content);
const hasEnv = /^\[env\]/m.test(content);

if (hasTools || hasEnv) {
  log("⚠ Your mise.toml already has [tools] and/or [env] sections.");
  log("  Add the following lines manually to your existing sections:");
  log("");
  if (hasTools) {
    log("  # In your [tools] section:");
    log('  "cargo:tauri-cli" = "2"');
    log('  bun               = "latest"');
    log('  xcodegen          = {version = "latest", os = ["macos"]}');
    log('  ruby              = {version = "3.3",    os = ["macos"]}');
    log('  java              = "temurin-17.0.18+8"');
    log("");
  }
  if (hasEnv) {
    log("  # In your [env] section:");
    log('  ANDROID_HOME = "{{env.HOME}}/.android-sdk"');
    log('  NDK_HOME = "{{env.HOME}}/.android-sdk/ndk/27.2.12479018"');
    log('  JAVA_HOME = "{{env.HOME}}/.local/share/mise/installs/java/temurin-17.0.18+8"');
    log('  _.path = ["{{env.HOME}}/.android-sdk/platform-tools", "{{env.HOME}}/.android-sdk/emulator", "{{env.HOME}}/.android-sdk/cmdline-tools/latest/bin"]');
    log("");
  }
  process.exit(0);
}

const block = `
# ── Added by: mise run init ──────────────────────────────────────────────────

# utm-dev tools — added by mise run init
[tools]
"cargo:tauri-cli" = "2"
bun               = "latest"
xcodegen          = {version = "latest", os = ["macos"]}
ruby              = {version = "3.3",    os = ["macos"]}
java              = "temurin-17.0.18+8"

# utm-dev env — Android SDK paths (installed by mise run setup)
[env]
ANDROID_HOME = "{{env.HOME}}/.android-sdk"
NDK_HOME = "{{env.HOME}}/.android-sdk/ndk/27.2.12479018"
JAVA_HOME = "{{env.HOME}}/.local/share/mise/installs/java/temurin-17.0.18+8"
_.path = ["{{env.HOME}}/.android-sdk/platform-tools", "{{env.HOME}}/.android-sdk/emulator", "{{env.HOME}}/.android-sdk/cmdline-tools/latest/bin"]
`;

appendFileSync(miseToml, block);

log(`✓ Added [tools] and [env] to ${miseToml}`);
log("");
log("Next:");
log("  mise install      # Install tools");
log("  mise run setup    # Install SDKs");
