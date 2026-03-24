#!/usr/bin/env bun

// #MISE description="Install UTM + download Windows VM + start + wait"
// #MISE alias="vm-up"

// Fully unattended — works from nothing, no prompts, no GUI interaction.
// Works both locally and when pulled as a remote mise task include.

import { $ } from "bun";
import { existsSync, readFileSync, writeFileSync, mkdirSync, statSync } from "fs";
import { join, dirname } from "path";
import {
  PROJECT_DIR, LOGDIR, STATEFILE, UTMCTL,
  SSH_PORT, RDP_PORT, WINRM_PORT, VM_USER, VM_PASS,
  info, ok, die, log, timestamp,
} from "../_lib.ts";

const LOG = "vm-up.log";
mkdirSync(dirname(STATEFILE), { recursive: true });
log(`── ${timestamp()} ──`, LOG);

const BOX_NAME = "windows-11";

let VM_UUID = "";
let VM_DISPLAY_NAME = "";

// Load existing state
if (existsSync(STATEFILE)) {
  const content = readFileSync(STATEFILE, "utf-8");
  VM_UUID = content.match(/VM_UUID="([^"]*)"/)?.[1] ?? "";
  VM_DISPLAY_NAME = content.match(/VM_DISPLAY_NAME="([^"]*)"/)?.[1] ?? "";
}

function saveState() {
  writeFileSync(STATEFILE, `VM_UUID="${VM_UUID}"\nVM_DISPLAY_NAME="${VM_DISPLAY_NAME}"\n`);
}

async function waitForUtmctl(maxSeconds = 30): Promise<boolean> {
  for (let i = 0; i < maxSeconds; i++) {
    const r = await $`${UTMCTL} list`.quiet().nothrow();
    if (r.exitCode === 0) return true;
    await Bun.sleep(1000);
  }
  return false;
}

async function getFirstVm(): Promise<string> {
  const r = await $`${UTMCTL} list`.quiet().nothrow();
  const lines = (r.stdout?.toString() ?? "").split("\n").filter((l) => l.trim() && !l.startsWith("UUID"));
  return lines[0]?.trim() ?? "";
}

function parseVmLine(line: string): { uuid: string; status: string; name: string } {
  const parts = line.split(/\s+/);
  return {
    uuid: parts[0] ?? "",
    status: parts[1] ?? "",
    name: parts.slice(2).join(" ").trim(),
  };
}

// ── 0. Kill any existing UTM to start clean ───────────────────────────────

await $`osascript -e 'tell application "UTM" to quit'`.quiet().nothrow();
await $`killall UTM`.quiet().nothrow();
await Bun.sleep(1000);

// ── 1. Install UTM ────────────────────────────────────────────────────────

if (existsSync(UTMCTL)) {
  ok("UTM installed");
} else {
  info("Installing UTM via brew...", LOG);
  await $`HOMEBREW_NO_AUTO_UPDATE=1 brew install --cask utm < /dev/null`;
  if (!existsSync(UTMCTL)) die("UTM install failed");
  ok("UTM installed", LOG);
}

// Suppress What's New dialog before first launch
const plistPath = "/Applications/UTM.app/Contents/Info.plist";
if (existsSync(plistPath)) {
  const verResult = await $`/usr/libexec/PlistBuddy -c "Print :CFBundleShortVersionString" ${plistPath}`.quiet().nothrow();
  const utmVersion = verResult.stdout.toString().trim();
  if (utmVersion) {
    const containerPrefs = `${process.env.HOME}/Library/Containers/com.utmapp.UTM/Data/Library/Preferences`;
    mkdirSync(containerPrefs, { recursive: true });
    await $`defaults write ${containerPrefs}/com.utmapp.UTM ReleaseNotesLastVersion -string ${utmVersion}`;
    ok(`Suppressed What's New dialog (v${utmVersion})`, LOG);
  }
}

// Launch UTM in background
await $`open -g /Applications/UTM.app`;
info("Waiting for UTM...", LOG);
if (!(await waitForUtmctl(30))) die("UTM did not become ready after 30s");
ok("UTM ready", LOG);

// ── 2. Find or download + import VM ───────────────────────────────────────

let vmExists = false;

// Check by saved UUID
if (VM_UUID) {
  const list = await $`${UTMCTL} list`.quiet().nothrow();
  if (list.stdout.toString().includes(VM_UUID)) {
    vmExists = true;
    ok(`VM exists (${VM_DISPLAY_NAME || "unknown"})`, LOG);
  }
}

// Check by scanning
if (!vmExists) {
  const vmLine = await getFirstVm();
  if (vmLine) {
    const parsed = parseVmLine(vmLine);
    VM_UUID = parsed.uuid;
    VM_DISPLAY_NAME = parsed.name;
    vmExists = true;
    saveState();
    ok(`VM exists (${VM_DISPLAY_NAME})`, LOG);
  }
}

// Download and import
if (!vmExists) {
  const boxCacheDir = `${process.env.HOME}/.cache/utm-dev`;
  const boxArch = "arm64";
  mkdirSync(boxCacheDir, { recursive: true });

  info("Fetching box version...", LOG);
  const versionsApi = `https://api.cloud.hashicorp.com/vagrant/2022-09-30/registry/utm/box/${BOX_NAME}/versions`;
  const versionsRes = await fetch(versionsApi);
  if (!versionsRes.ok) die("Cannot reach Vagrant API");
  const versionsJson = await versionsRes.text();
  const boxVersion = versionsJson.match(/"name":"([^"]*)"/)?.[1];
  if (!boxVersion) die("Cannot parse box version");
  info(`Latest box version: ${boxVersion}`, LOG);

  const boxFile = `${boxCacheDir}/${BOX_NAME}_${boxVersion}_${boxArch}.box`;

  if (existsSync(boxFile)) {
    ok("Box cached (skipping download)", LOG);
  } else {
    // Clean old boxes
    await $`rm -f ${boxCacheDir}/${BOX_NAME}_*.box`.nothrow();

    info("Downloading box (~6 GB) — this takes a while...", LOG);
    const downloadApi = `https://api.cloud.hashicorp.com/vagrant/2022-09-30/registry/utm/box/${BOX_NAME}/version/${boxVersion}/provider/utm/architecture/${boxArch}/download`;
    const downloadRes = await fetch(downloadApi);
    if (!downloadRes.ok) die("Cannot fetch download URL");
    const downloadJson = await downloadRes.text();
    const boxUrl = downloadJson.match(/"url":"([^"]*)"/)?.[1];
    if (!boxUrl) die("Cannot parse download URL");

    await $`curl -fL --progress-bar -o ${boxFile}.partial ${boxUrl}`;

    const fileSize = statSync(`${boxFile}.partial`).size;
    if (fileSize < 1_000_000_000) {
      await $`rm -f ${boxFile}.partial`;
      die(`Download too small (${fileSize} bytes)`);
    }
    await $`mv ${boxFile}.partial ${boxFile}`;
    ok(`Box downloaded (${boxVersion})`, LOG);
  }

  info("Extracting box...", LOG);
  const tmpdir = (await $`mktemp -d`.quiet()).stdout.toString().trim();
  await $`tar -xf ${boxFile} -C ${tmpdir}`;
  const utmFolder = (
    await $`find ${tmpdir} -type d -name "*.utm"`.quiet()
  ).stdout.toString().trim().split("\n")[0];
  if (!utmFolder) {
    await $`rm -rf ${tmpdir}`;
    die("No .utm folder in box");
  }

  info("Importing VM into UTM...", LOG);
  const importResult = await $`osascript -e ${"tell application \"UTM\" to import new virtual machine from POSIX file \"" + utmFolder + "\""}`
    .quiet()
    .nothrow();
  await $`rm -rf ${tmpdir}`;
  if (importResult.exitCode !== 0) die("Import failed");

  // Wait for import to register
  info("Waiting for import...", LOG);
  let vmLine = "";
  for (let i = 0; i < 15; i++) {
    vmLine = await getFirstVm();
    if (vmLine) break;
    await Bun.sleep(2000);
  }
  if (!vmLine) die("Import failed — no VM found after 30s");
  const parsed = parseVmLine(vmLine);
  VM_UUID = parsed.uuid;
  VM_DISPLAY_NAME = parsed.name;
  saveState();
  ok(`VM imported (${VM_DISPLAY_NAME})`, LOG);
}

// ── 3. Configure network ──────────────────────────────────────────────────
// Uses AppleScript to read config, set port forwards on emulated NIC, write back.

const vmStatusLine = await $`${UTMCTL} list`.quiet().nothrow();
const currentStatus = vmStatusLine.stdout.toString().split("\n")
  .find((l) => l.includes(VM_UUID))?.split(/\s+/)[1];

if (currentStatus === "started") {
  info("Stopping VM to configure network...", LOG);
  await $`${UTMCTL} stop ${VM_DISPLAY_NAME}`.quiet().nothrow();
  await Bun.sleep(8000);
}

// Find the emulated NIC index
const emulatedResult = await $`osascript -e ${`
  tell application "UTM"
    set vm to virtual machine id "${VM_UUID}"
    set cfg to configuration of vm
    set nis to network interfaces of cfg
    repeat with ni in nis
      if mode of ni is emulated then
        return index of ni
      end if
    end repeat
    return -1
  end tell
`}`.quiet().nothrow();

const emulatedIndex = emulatedResult.stdout.toString().trim();
if (emulatedIndex === "-1" || !emulatedIndex) die("No emulated network interface found on VM");

info(`Configuring port forwards on NIC index ${emulatedIndex}...`, LOG);

// Inline AppleScript — based on vagrant_utm's approach: read config, mutate, write back.
const portForwardScript = `
on run argv
  set vmID to item 1 of argv
  set portForwardRules to {}
  repeat with i from 2 to count of argv by 3
    set indexNumber to (item (i + 1) of argv) as integer
    set ruleArg to item (i + 2) of argv
    set AppleScript's text item delimiters to ","
    set ruleComponents to text items of ruleArg
    set portForwardRule to {indexVal:indexNumber, protocolVal:item 1 of ruleComponents, guestAddress:item 2 of ruleComponents, guestPort:item 3 of ruleComponents, hostAddress:item 4 of ruleComponents, hostPort:item 5 of ruleComponents}
    set end of portForwardRules to portForwardRule
  end repeat
  tell application "UTM"
    set vm to virtual machine id vmID
    set config to configuration of vm
    set networkInterfaces to network interfaces of config
    repeat with anInterface in networkInterfaces
      set netIfIndex to index of anInterface
      if mode of anInterface is emulated then
        set port forwards of anInterface to {}
      end if
      repeat with portForwardRule in portForwardRules
        if (indexVal of portForwardRule) as integer is netIfIndex then
          set portForwards to port forwards of anInterface
          set newPortForward to {protocol:(protocolVal of portForwardRule), guest address:(guestAddress of portForwardRule), guest port:(guestPort of portForwardRule), host address:(hostAddress of portForwardRule), host port:(hostPort of portForwardRule)}
          copy newPortForward to the end of portForwards
          set port forwards of anInterface to portForwards
        end if
      end repeat
    end repeat
    update configuration of vm with config
  end tell
end run
`;

const pfResult = await $`osascript - ${VM_UUID} --index ${emulatedIndex} ${"TcPp,,22,127.0.0.1," + SSH_PORT} --index ${emulatedIndex} ${"TcPp,,3389,127.0.0.1," + RDP_PORT} --index ${emulatedIndex} ${"TcPp,,5985,127.0.0.1," + WINRM_PORT} << ${portForwardScript}`.quiet().nothrow();
if (pfResult.exitCode !== 0) die("Failed to configure port forwards");

ok(`Network: SSH:${SSH_PORT} RDP:${RDP_PORT} WinRM:${WINRM_PORT}`, LOG);

// ── 4. Start VM ───────────────────────────────────────────────────────────

info("Starting VM...", LOG);
let started = false;
for (let attempt = 1; attempt <= 3; attempt++) {
  const r = await $`${UTMCTL} start ${VM_DISPLAY_NAME}`.quiet().nothrow();
  if (r.exitCode === 0) {
    started = true;
    ok("VM started", LOG);
    break;
  }
  log(`  retry ${attempt}/3...`, LOG);
  await Bun.sleep(5000);
}
if (!started) die("Failed to start VM after 3 attempts");

// ── 5. Wait for Windows boot ──────────────────────────────────────────────

info("Waiting for Windows to boot (up to 5 min)...", LOG);
const timeout = 300;
let elapsed = 0;
while (elapsed < timeout) {
  try {
    await fetch(`http://127.0.0.1:${WINRM_PORT}/wsman`, {
      signal: AbortSignal.timeout(2000),
    });
    ok(`Windows ready (${elapsed}s)`, LOG);
    break;
  } catch {}
  await Bun.sleep(5000);
  elapsed += 5;
  if (elapsed % 30 === 0) log(`  still booting... (${elapsed}s)`, LOG);
}
if (elapsed >= timeout) die(`Timeout waiting for Windows (${timeout}s)`);

// ── 6. Bootstrap SSH + Rust ───────────────────────────────────────────────

const taskDir = dirname(new URL(import.meta.url).pathname);
const bootstrapPath = join(taskDir, "bootstrap");
if (existsSync(bootstrapPath)) {
  await $`${bootstrapPath}`;
} else {
  info("Skipping bootstrap (task not found — run mise run vm:bootstrap manually)", LOG);
}

log("", LOG);
ok("Done", LOG);
log(`  RDP: localhost:${RDP_PORT} (${VM_USER}/${VM_PASS})`, LOG);
log(`  SSH: sshpass -p ${VM_PASS} ssh -p ${SSH_PORT} ${VM_USER}@127.0.0.1`, LOG);
