#!/usr/bin/env bun

//MISE description="Start a Windows VM: vm:up [build|test]"
//MISE alias="vm-up"

// Starts a VM that was set up by `mise run setup`.
// If the VM hasn't been set up yet, tells you to run setup.

import {
  parseVMArg, getProfile, loadState, hasState,
  ok, die, log, timestamp,
} from "../_lib.ts";
import { ensureUtm, startVm, waitForBoot } from "../_utm.ts";

const LOG = "vm-up.log";
log(`── ${timestamp()} ──`, LOG);

const { vmName } = parseVMArg();
const profile = getProfile(vmName);

if (!hasState(vmName)) {
  die(`${vmName} VM not set up yet. Run: mise run setup`);
}

const { VM_DISPLAY_NAME } = loadState(vmName);

await ensureUtm(LOG);
await startVm(VM_DISPLAY_NAME, LOG);
await waitForBoot(profile.winrmPort, 300, LOG);

log("", LOG);
ok(`${vmName} VM ready`, LOG);
log(`  RDP: localhost:${profile.rdpPort} (${profile.user}/${profile.pass})`, LOG);
log(`  SSH: sshpass -p ${profile.pass} ssh -p ${profile.sshPort} ${profile.user}@127.0.0.1`, LOG);
