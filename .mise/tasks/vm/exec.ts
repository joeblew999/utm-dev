#!/usr/bin/env bun

//MISE description="Run a command inside the Windows VM via SSH"
//MISE alias="vm-exec"

import {
  ensureSshpass, checkSsh, ssh, info, log, timestamp,
} from "../_lib.ts";

const LOG = "vm-exec.log";
log(`── ${timestamp()} ──`, LOG);

const args = process.argv.slice(2);
if (args.length === 0) {
  console.log("Usage: mise vm:exec <command>");
  console.log("  mise vm:exec 'whoami'");
  console.log("  mise vm:exec 'cargo --version'");
  process.exit(1);
}

await ensureSshpass();
await checkSsh();

const cmd = args.join(" ");
info(cmd, LOG);
const result = await ssh(cmd);
process.exit(result.exitCode);
