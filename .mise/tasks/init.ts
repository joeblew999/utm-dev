#!/usr/bin/env bun

//MISE description="Add utm-dev tools and env to your project's mise.toml"
//MISE alias="i"

// Adds the [tools] and [env] blocks needed for Tauri cross-platform builds.
// Idempotent — safe to run multiple times.

import { existsSync, readFileSync, appendFileSync } from "fs";
import { join } from "path";

const miseToml = join(process.cwd(), "mise.toml");

if (!existsSync(miseToml)) {
  console.log(`✗ No mise.toml found in ${process.cwd()}`);
  console.log("  Create one first, or run from your project root.");
  process.exit(1);
}

const content = readFileSync(miseToml, "utf-8");

// Check if we already added our blocks
if (content.includes("# utm-dev tools")) {
  console.log("✓ Already initialised");
  process.exit(0);
}

const hasTools = /^\[tools\]/m.test(content);
const hasEnv = /^\[env\]/m.test(content);

if (hasTools || hasEnv) {
  console.log("⚠ Your mise.toml already has [tools] and/or [env] sections.");
  console.log("  Add the following lines manually to your existing sections:");
  console.log("");
  if (hasTools) {
    console.log("  # In your [tools] section:");
    console.log('  "cargo:tauri-cli" = {version = "2",      os = ["macos", "windows"]}');
    console.log('  bun               = "latest"');
    console.log('  xcodegen          = {version = "latest", os = ["macos"]}');
    console.log('  ruby              = {version = "3.3",    os = ["macos"]}');
    console.log('  java              = "temurin-17.0.18+8"');
    console.log("");
  }
  if (hasEnv) {
    console.log("  # In your [env] section:");
    console.log('  ANDROID_HOME = "{{env.HOME}}/.android-sdk"');
    console.log('  NDK_HOME = "{{env.HOME}}/.android-sdk/ndk/27.2.12479018"');
    console.log('  JAVA_HOME = "{{env.HOME}}/.local/share/mise/installs/java/temurin-17.0.18+8"');
    console.log("");
  }
  process.exit(0);
}

const block = `
# ── Added by: mise run init ──────────────────────────────────────────────────

# utm-dev tools — added by mise run init
[tools]
"cargo:tauri-cli" = {version = "2",      os = ["macos", "windows"]}
bun               = "latest"
xcodegen          = {version = "latest", os = ["macos"]}
ruby              = {version = "3.3",    os = ["macos"]}
java              = "temurin-17.0.18+8"

# utm-dev env — Android SDK paths (installed by mise run setup)
[env]
ANDROID_HOME = "{{env.HOME}}/.android-sdk"
NDK_HOME = "{{env.HOME}}/.android-sdk/ndk/27.2.12479018"
JAVA_HOME = "{{env.HOME}}/.local/share/mise/installs/java/temurin-17.0.18+8"
`;

appendFileSync(miseToml, block);

console.log(`✓ Added [tools] and [env] to ${miseToml}`);
console.log("");
console.log("Next:");
console.log("  mise install      # Install tools");
console.log("  mise run setup    # Install SDKs");
