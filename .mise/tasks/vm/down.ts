#!/usr/bin/env bun

//MISE description="Stop a Windows VM: vm:down [build|test]"
//MISE alias="vm-down"

import { $ } from "bun";
import {
  UTMCTL, parseVMArg, loadState, info, ok, die, log, timestamp,
} from "../_lib.ts";

const LOG = "vm-down.log";
log(`── ${timestamp()} ──`, LOG);

const { vmName } = parseVMArg();
const { VM_UUID, VM_DISPLAY_NAME } = loadState(vmName);

const list = await $`${UTMCTL} list`.quiet().nothrow();
if (list.exitCode !== 0) die("UTM not running");

const lines = list.stdout.toString();
const vmLine = lines.split("\n").find((l) => l.includes(VM_UUID));
if (!vmLine) die(`VM "${vmName}" not found (UUID: ${VM_UUID})`);

const status = vmLine.split(/\s+/)[1];

switch (status) {
  case "started":
    info(`Stopping '${VM_DISPLAY_NAME}' (${vmName})...`, LOG);
    await $`${UTMCTL} stop ${VM_DISPLAY_NAME}`;
    ok("Stopped", LOG);
    break;
  case "stopped":
    ok("Already stopped", LOG);
    break;
  default:
    die(`VM "${vmName}" not found (UUID: ${VM_UUID})`);
}
