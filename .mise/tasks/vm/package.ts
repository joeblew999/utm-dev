#!/usr/bin/env bun

//MISE description="Export a VM as a reusable Vagrant box: vm:package [build|test]"
//MISE alias="vm-package"

import { $ } from "bun";
import { mkdirSync, writeFileSync, existsSync } from "fs";
import { join } from "path";
import {
  PROJECT_DIR, UTMCTL,
  parseVMArg, loadState,
  info, ok, die, log, timestamp,
} from "../_lib.ts";

const LOG = "vm-package.log";
log(`── ${timestamp()} ──`, LOG);

const { vmName } = parseVMArg();
const { VM_UUID, VM_DISPLAY_NAME } = loadState(vmName);

// ── Stop VM if running ────────────────────────────────────────────────────

const list = await $`${UTMCTL} list`.quiet().nothrow();
const vmLine = (list.stdout?.toString() ?? "")
  .split("\n")
  .find((l) => l.includes(VM_UUID));

if (vmLine && vmLine.includes("started")) {
  info(`Stopping ${vmName} VM before export...`, LOG);
  await $`${UTMCTL} stop ${VM_DISPLAY_NAME}`.quiet().nothrow();
  await Bun.sleep(8000);
}

// ── Find the .utm bundle ──────────────────────────────────────────────────

const home = process.env.HOME!;
const utmBundle = join(
  home,
  "Library/Containers/com.utmapp.UTM/Data/Documents",
  `${VM_DISPLAY_NAME}.utm`,
);
if (!existsSync(utmBundle)) die(`VM bundle not found at ${utmBundle}`);

const duResult = await $`du -sh ${utmBundle}`.quiet();
const bundleSize = duResult.stdout.toString().split("\t")[0];
info(`VM bundle: ${utmBundle} (${bundleSize})`, LOG);

// ── Package as .box ───────────────────────────────────────────────────────

const boxOutputDir = join(PROJECT_DIR, ".build", "boxes");
mkdirSync(boxOutputDir, { recursive: true });
const boxFile = join(boxOutputDir, `windows-11-${vmName}_arm64.box`);

const tmpdir = await $`mktemp -d`.quiet();
const tmpdirPath = tmpdir.stdout.toString().trim();

try {
  info("Creating metadata.json...", LOG);
  writeFileSync(join(tmpdirPath, "metadata.json"), JSON.stringify({ provider: "utm" }));

  info("Copying VM bundle (this may take a minute)...", LOG);
  await $`cp -a ${utmBundle} ${join(tmpdirPath, "box.utm")}`;

  info("Creating .box archive...", LOG);
  await $`tar -cf ${boxFile} -C ${tmpdirPath} metadata.json box.utm`;
} finally {
  await $`rm -rf ${tmpdirPath}`.nothrow();
}

const boxDu = await $`du -sh ${boxFile}`.quiet();
const boxSize = boxDu.stdout.toString().split("\t")[0];

log("", LOG);
ok(`Box created: ${boxFile} (${boxSize})`, LOG);
log("", LOG);
log("To publish to Vagrant Cloud:", LOG);
log("  1. Create an account at https://app.vagrantup.com", LOG);
log(`  2. Create a box: joeblew999/windows-11-${vmName}`, LOG);
log(`  3. Upload: vagrant cloud publish joeblew999/windows-11-${vmName} 1.0.0 utm ${boxFile}`, LOG);
