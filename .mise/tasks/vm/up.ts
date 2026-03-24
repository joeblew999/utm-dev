#!/usr/bin/env bun

//MISE description="Start a VM (imports + bootstraps on first run)"
//MISE alias="vm-up"
//MISE hide=true

// If the VM hasn't been imported yet, downloads the box, imports it,
// configures networking, and runs bootstrap — then starts it.
// After first run, just starts the existing VM.

import { existsSync } from "fs";
import { dirname, join } from "path";
import { $ } from "bun";
import {
  parseVMArg, getProfile, loadState, hasState,
  ok, info, log, timestamp,
} from "../_lib.ts";
import { ensureUtm, importBox, configureNetwork, configureResources, stopVm, startVm, waitForBoot } from "../_utm.ts";

const LOG = "vm-up.log";
log(`── ${timestamp()} ──`, LOG);

const { vmName } = parseVMArg();
const profile = getProfile(vmName);

await ensureUtm(LOG);

// ── First run: import box + configure network + bootstrap ────────────────

if (!hasState(vmName)) {
  info(`${vmName} VM not found — setting up...`, LOG);

  // Install sshpass (needed for SSH after bootstrap)
  if ((await $`command -v sshpass`.quiet().nothrow()).exitCode !== 0) {
    info("Installing sshpass...", LOG);
    await $`HOMEBREW_NO_AUTO_UPDATE=1 brew install hudochenkov/sshpass/sshpass < /dev/null 2>/dev/null`.nothrow();
  }

  // Import box
  const { uuid, displayName } = await importBox(profile, vmName, LOG);

  // Configure network + resources (VM must be stopped)
  await stopVm(displayName, LOG);
  await configureNetwork(uuid, profile, LOG);
  await configureResources(uuid, profile.memoryMiB, profile.cpuCores, LOG);

  // Bootstrap (start VM, run bootstrap, stop, then start again below)
  if (profile.bootstrap) {
    await startVm(displayName, LOG);
    await waitForBoot(profile, 300, LOG);

    const taskDir = dirname(new URL(import.meta.url).pathname);
    const bootstrapScript = profile.os === "linux" ? "_bootstrapLinux.ts" : "_bootstrap.ts";
    const bootstrapPath = join(taskDir, "..", bootstrapScript);
    if (existsSync(bootstrapPath)) {
      await $`bun ${bootstrapPath} ${vmName}`;
    }

    await stopVm(displayName, LOG);
  }

  ok(`${vmName} VM setup complete`, LOG);
}

// ── Start VM ─────────────────────────────────────────────────────────────

const { VM_DISPLAY_NAME } = loadState(vmName);
await startVm(VM_DISPLAY_NAME, LOG);
await waitForBoot(profile, 300, LOG);

log("", LOG);
ok(`${vmName} VM ready`, LOG);
if (profile.os === "windows") {
  log(`  RDP: localhost:${profile.rdpPort} (${profile.user}/${profile.pass})`, LOG);
}
log(`  SSH: sshpass -p ${profile.pass} ssh -p ${profile.sshPort} ${profile.user}@127.0.0.1`, LOG);
