#!/usr/bin/env bun

//MISE description="Sync project files into Windows VM"
//MISE alias="vm-sync"
//MISE sources=["src-tauri/**/*", "ui/**/*", "Cargo.toml", "Cargo.lock", "mise.toml", "package.json"]
//MISE outputs=[".mise/state/last-sync"]

import { $ } from "bun";
import { mkdirSync, writeFileSync } from "fs";
import { join } from "path";
import {
  PROJECT_DIR, PROJECT_NAME, VM_USER, VM_HOME,
  ensureSshpass, checkSsh, ssh, scp, info, ok, log, timestamp,
} from "../_lib.ts";

const LOG = "vm-sync.log";
log(`── ${timestamp()} ──`, LOG);

await ensureSshpass();
await checkSsh();

info(`Syncing ${PROJECT_NAME} to VM...`, LOG);

const tmptar = `/tmp/vm-sync-${Date.now()}.tar.gz`;

try {
  await $`tar -czf ${tmptar} --exclude=target --exclude=node_modules --exclude=.git --exclude=.mise/logs --exclude=.mise/state --exclude=.build --exclude=.gradle -C ${PROJECT_DIR} .`;

  await scp(tmptar, `${VM_USER}@127.0.0.1:sync.tar.gz`);

  const vmProjectDir = `${VM_HOME}\\${PROJECT_NAME}`;
  await ssh(
    `if not exist "${vmProjectDir}" mkdir "${vmProjectDir}" && cd "${vmProjectDir}" && tar -xzf "%USERPROFILE%\\sync.tar.gz" && del "%USERPROFILE%\\sync.tar.gz"`,
  );

  mkdirSync(join(PROJECT_DIR, ".mise", "state"), { recursive: true });
  writeFileSync(join(PROJECT_DIR, ".mise", "state", "last-sync"), new Date().toISOString());

  ok(`Synced to ${vmProjectDir}`, LOG);
} finally {
  await $`rm -f ${tmptar}`.nothrow();
}
