#!/usr/bin/env bun

//MISE description="Run a command inside a Windows VM via SSH: vm:exec [build|test] <cmd>"
//MISE alias="vm-exec"

import {
  parseVMArg, getProfile, ensureSshpass, checkSsh, ssh, info, log, timestamp,
} from "../_lib.ts";

const LOG = "vm-exec.log";
log(`── ${timestamp()} ──`, LOG);

const { vmName, rest } = parseVMArg();
const profile = getProfile(vmName);

if (rest.length === 0) {
  console.log("Usage: mise vm:exec [build|test] <command>");
  console.log("  mise vm:exec 'whoami'              # default: build VM");
  console.log("  mise vm:exec build 'cargo --version'");
  console.log("  mise vm:exec test 'dir'");
  process.exit(1);
}

await ensureSshpass();
await checkSsh(profile);

const cmd = rest.join(" ");
info(cmd, LOG);
const result = await ssh(profile, cmd);
process.exit(result.exitCode);
