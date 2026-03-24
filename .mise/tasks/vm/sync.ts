#!/usr/bin/env bun

//MISE description="Sync project files into a Windows VM: vm:sync [build|test]"
//MISE alias="vm-sync"
//MISE sources=["src-tauri/**/*", "ui/**/*", "Cargo.toml", "Cargo.lock", "mise.toml", "package.json"]
//MISE outputs=[".mise/state/last-sync"]

import { $ } from "bun";
import { mkdirSync, writeFileSync } from "fs";
import { join } from "path";
import {
  PROJECT_DIR, PROJECT_NAME,
  parseVMArg, getProfile, vmHome,
  ensureSshpass, checkSsh, ssh, scp, info, ok, log, timestamp,
} from "../_lib.ts";

const LOG = "vm-sync.log";
log(`── ${timestamp()} ──`, LOG);

const { vmName } = parseVMArg();
const profile = getProfile(vmName);

await ensureSshpass();
await checkSsh(profile);

info(`Syncing ${PROJECT_NAME} to ${vmName} VM...`, LOG);

const tmptar = `/tmp/vm-sync-${Date.now()}.tar.gz`;

try {
  await $`tar -czf ${tmptar} --exclude=target --exclude=node_modules --exclude=.git --exclude=.mise/logs --exclude=.mise/state --exclude=.build --exclude=.gradle -C ${PROJECT_DIR} .`;

  await scp(profile, tmptar, `${profile.user}@127.0.0.1:sync.tar.gz`);

  const home = vmHome(profile);
  const vmProjectDir = `${home}\\${PROJECT_NAME}`;
  await ssh(
    profile,
    `if not exist "${vmProjectDir}" mkdir "${vmProjectDir}" && cd "${vmProjectDir}" && tar -xzf "%USERPROFILE%\\sync.tar.gz" && del "%USERPROFILE%\\sync.tar.gz"`,
  );

  mkdirSync(join(PROJECT_DIR, ".mise", "state"), { recursive: true });
  writeFileSync(join(PROJECT_DIR, ".mise", "state", "last-sync"), new Date().toISOString());

  ok(`Synced to ${vmProjectDir}`, LOG);
} finally {
  await $`rm -f ${tmptar}`.nothrow();
}
