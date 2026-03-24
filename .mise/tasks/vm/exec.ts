#!/usr/bin/env bun

//MISE description="Run a command in a VM via SSH"
//MISE alias="vm-exec"
//MISE hide=true

import {
  parseVMArg, getProfile, ensureSshpass, checkSsh, ssh, info, log, timestamp,
} from "../_lib.ts";

const LOG = "vm-exec.log";
log(`── ${timestamp()} ──`, LOG);

const { vmName, rest } = parseVMArg();
const profile = getProfile(vmName);

if (rest.length === 0) {
  log("Usage: mise vm:exec [windows-build|windows-test|linux-build|linux-test] <command>");
  log("  mise vm:exec 'whoami'                     # default: windows-build VM");
  log("  mise vm:exec windows-build 'cargo --version'");
  log("  mise vm:exec windows-test 'dir'");
  process.exit(1);
}

await ensureSshpass();
await checkSsh(profile);

const cmd = rest.join(" ");
info(cmd, LOG);
const result = await ssh(profile, cmd);
process.exit(result.exitCode);
