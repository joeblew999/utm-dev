#!/usr/bin/env bun

// #MISE description="Stop the Windows VM"
// #MISE alias="vm-down"

import { $ } from "bun";
import {
  LOGDIR, UTMCTL, loadState, info, ok, die, log, timestamp,
} from "../_lib.ts";

const LOG = "vm-down.log";
log(`── ${timestamp()} ──`, LOG);

const { VM_UUID, VM_DISPLAY_NAME } = loadState();

const list = await $`${UTMCTL} list`.quiet().nothrow();
if (list.exitCode !== 0) die("UTM not running");

const lines = list.stdout.toString();
const vmLine = lines.split("\n").find((l) => l.includes(VM_UUID));
if (!vmLine) die(`VM not found (UUID: ${VM_UUID})`);

const status = vmLine.split(/\s+/)[1];

switch (status) {
  case "started":
    info(`Stopping '${VM_DISPLAY_NAME}'...`, LOG);
    await $`${UTMCTL} stop ${VM_DISPLAY_NAME}`;
    ok("Stopped", LOG);
    break;
  case "stopped":
    ok("Already stopped", LOG);
    break;
  default:
    die(`VM not found (UUID: ${VM_UUID})`);
}
