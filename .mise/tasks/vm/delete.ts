#!/usr/bin/env bun

// #MISE description="Delete VM and/or UTM: vm:delete vm | utm | all"

import { $ } from "bun";
import { existsSync, readFileSync, unlinkSync, rmSync } from "fs";
import { join } from "path";
import {
  PROJECT_DIR, LOGDIR, STATEFILE, UTMCTL, info, ok, die, log, timestamp,
} from "../_lib.ts";

const LOG = "vm-delete.log";
log(`── ${timestamp()} ──`, LOG);

const arg = process.argv[2];
if (!arg) {
  console.log("Usage: mise vm:delete vm|utm|all");
  process.exit(1);
}

async function deleteVm() {
  const utmExists = await $`test -x ${UTMCTL}`.quiet().nothrow();
  if (utmExists.exitCode !== 0) {
    ok("UTM not installed, no VM", LOG);
    return;
  }

  // Ensure UTM is running so utmctl works
  const listCheck = await $`${UTMCTL} list`.quiet().nothrow();
  if (listCheck.exitCode !== 0) {
    await $`open -g /Applications/UTM.app`.nothrow();
    for (let i = 0; i < 15; i++) {
      const r = await $`${UTMCTL} list`.quiet().nothrow();
      if (r.exitCode === 0) break;
      await Bun.sleep(1000);
    }
  }

  const list = await $`${UTMCTL} list`.quiet().nothrow();
  const lines = (list.stdout?.toString() ?? "")
    .split("\n")
    .filter((l) => l.trim() && !l.startsWith("UUID"));

  if (lines.length === 0) {
    ok("No VMs", LOG);
    return;
  }

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
    const del = await $`${UTMCTL} delete ${name}`.quiet().nothrow();
    if (del.exitCode !== 0) {
      await $`osascript -e 'tell application "UTM" to delete virtual machine id "${uuid}"'`
        .quiet()
        .nothrow();
    }
    ok(`Deleted '${name}'`, LOG);
  }

  if (existsSync(STATEFILE)) unlinkSync(STATEFILE);
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

function cleanData() {
  info("Cleaning UTM app data (preserving box cache)...", LOG);
  const home = process.env.HOME!;
  rmSync(join(home, "Library/Containers/com.utmapp.UTM"), { recursive: true, force: true });
  // Clean Group Containers matching UTM
  try {
    const groupDir = join(home, "Library/Group Containers");
    const { readdirSync } = require("fs");
    for (const d of readdirSync(groupDir)) {
      if (d.includes("com.utmapp.UTM")) {
        rmSync(join(groupDir, d), { recursive: true, force: true });
      }
    }
  } catch {}
  if (existsSync(STATEFILE)) unlinkSync(STATEFILE);
  ok("App data cleaned (box cache kept at ~/.cache/utm-dev/)", LOG);
}

switch (arg) {
  case "vm":
    await deleteVm();
    break;
  case "utm":
    await deleteVm();
    await deleteUtm();
    break;
  case "all":
    await deleteVm();
    await deleteUtm();
    cleanData();
    break;
  default:
    console.log("Usage: mise vm:delete vm|utm|all");
    process.exit(1);
}

log("Rebuild: mise vm:up", LOG);
