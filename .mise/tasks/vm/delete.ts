#!/usr/bin/env bun

//MISE description="Delete a VM or UTM"
//MISE alias="vm-delete"
//MISE hide=true

import { $ } from "bun";
import { readdirSync } from "fs";
import { join } from "path";
import {
  UTMCTL, VM_PROFILES,
  loadState, clearState, hasState,
  info, ok, log, timestamp,
} from "../_lib.ts";

const LOG = "vm-delete.log";
log(`── ${timestamp()} ──`, LOG);

const arg = process.argv[2];
if (!arg) {
  log("Usage: mise vm:delete windows-build|windows-test|linux-build|linux-test|utm|all");
  process.exit(1);
}

async function ensureUtmRunning() {
  const check = await $`${UTMCTL} list`.quiet().nothrow();
  if (check.exitCode === 0) return;
  await $`open -g /Applications/UTM.app`.nothrow();
  for (let i = 0; i < 15; i++) {
    const r = await $`${UTMCTL} list`.quiet().nothrow();
    if (r.exitCode === 0) return;
    await Bun.sleep(1000);
  }
}

async function deleteVmByName(vmName: string) {
  if (!hasState(vmName)) {
    ok(`No ${vmName} VM`, LOG);
    return;
  }

  const utmExists = await $`test -x ${UTMCTL}`.quiet().nothrow();
  if (utmExists.exitCode !== 0) {
    ok("UTM not installed, no VM", LOG);
    clearState(vmName);
    return;
  }

  await ensureUtmRunning();

  const { VM_UUID, VM_DISPLAY_NAME } = loadState(vmName);

  const list = await $`${UTMCTL} list`.quiet().nothrow();
  const vmLine = (list.stdout?.toString() ?? "")
    .split("\n")
    .find((l) => l.includes(VM_UUID));

  if (!vmLine) {
    ok(`${vmName} VM not in UTM (already removed)`, LOG);
    clearState(vmName);
    return;
  }

  const status = vmLine.split(/\s+/)[1];
  if (status === "started") {
    info(`Stopping '${VM_DISPLAY_NAME}'...`, LOG);
    await $`${UTMCTL} stop ${VM_DISPLAY_NAME}`.quiet().nothrow();
    await Bun.sleep(5000);
  }

  info(`Deleting '${VM_DISPLAY_NAME}' (${vmName})...`, LOG);
  const del = await $`${UTMCTL} delete ${VM_DISPLAY_NAME}`.quiet().nothrow();
  if (del.exitCode !== 0) {
    await $`osascript -e 'tell application "UTM" to delete virtual machine id "${VM_UUID}"'`
      .quiet()
      .nothrow();
  }
  ok(`Deleted '${VM_DISPLAY_NAME}'`, LOG);
  clearState(vmName);
}

async function deleteAllVms() {
  for (const vmName of Object.keys(VM_PROFILES)) {
    await deleteVmByName(vmName);
  }

  // Also clean up any VMs not tracked by state files
  const utmExists = await $`test -x ${UTMCTL}`.quiet().nothrow();
  if (utmExists.exitCode !== 0) return;

  await ensureUtmRunning();
  const list = await $`${UTMCTL} list`.quiet().nothrow();
  const lines = (list.stdout?.toString() ?? "")
    .split("\n")
    .filter((l) => l.trim() && !l.startsWith("UUID"));

  for (const line of lines) {
    const parts = line.trim().split(/\s+/);
    const uuid = parts[0];
    const status = parts[1];
    const name = parts.slice(2).join(" ");
    if (!uuid) continue;

    if (status === "started") {
      info(`Stopping '${name}'...`, LOG);
      await $`${UTMCTL} stop ${name}`.quiet().nothrow();
      await Bun.sleep(5000);
    }
    info(`Deleting '${name}'...`, LOG);
    await $`${UTMCTL} delete ${name}`.quiet().nothrow();
    ok(`Deleted '${name}'`, LOG);
  }
}

async function deleteUtm() {
  info("Quitting UTM...", LOG);
  await $`osascript -e 'tell application "UTM" to quit'`.quiet().nothrow();
  await $`killall -9 UTM`.quiet().nothrow();
  await Bun.sleep(2000);

  info("Uninstalling UTM...", LOG);
  const brew = await $`HOMEBREW_NO_AUTO_UPDATE=1 brew uninstall --cask utm < /dev/null`.quiet().nothrow();
  if (brew.exitCode !== 0) {
    await $`rm -rf /Applications/UTM.app`.quiet().nothrow();
  }
  ok("UTM uninstalled", LOG);
}

async function cleanData() {
  info("Cleaning UTM app data (preserving box cache)...", LOG);
  const home = process.env.HOME!;
  // macOS protects container dirs — use rm command which handles it better
  await $`rm -rf ${join(home, "Library/Containers/com.utmapp.UTM")}`.quiet().nothrow();
  try {
    const groupDir = join(home, "Library/Group Containers");
    for (const d of readdirSync(groupDir)) {
      if (d.includes("com.utmapp.UTM")) {
        await $`rm -rf ${join(groupDir, d)}`.quiet().nothrow();
      }
    }
  } catch {}
  for (const vmName of Object.keys(VM_PROFILES)) {
    clearState(vmName);
  }
  ok("App data cleaned (box cache kept at ~/.cache/utm-dev/)", LOG);
}

switch (arg) {
  case "windows-build":
  case "windows-test":
  case "linux-build":
  case "linux-test":
    await deleteVmByName(arg);
    break;
  case "utm":
    await deleteAllVms();
    await deleteUtm();
    break;
  case "all":
    await deleteAllVms();
    await deleteUtm();
    await cleanData();
    break;
  default:
    log("Usage: mise vm:delete windows-build|windows-test|linux-build|linux-test|utm|all");
    process.exit(1);
}

log("Rebuild: mise vm:up [windows-build|windows-test|linux-build|linux-test]", LOG);
