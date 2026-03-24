// UTM operations: install, import box, network config, start/stop, wait for boot.
// Extracted from vm/up.ts so setup.ts and vm/up.ts can both use these.

import { $ } from "bun";
import { existsSync, writeFileSync, mkdirSync, statSync } from "fs";
import { VMProfile, UTMCTL, saveState, info, ok, die, log } from "./_lib.ts";

// ── UTM app lifecycle ────────────────────────────────────────────────────────

/** Install UTM via brew if not present, launch it, wait for utmctl. */
export async function ensureUtm(logFile?: string): Promise<void> {
  const running = await $`${UTMCTL} list`.quiet().nothrow();
  if (running.exitCode === 0) return;

  // Install if needed
  if (!existsSync(UTMCTL)) {
    info("Installing UTM via brew...", logFile);
    await $`HOMEBREW_NO_AUTO_UPDATE=1 brew install --cask utm < /dev/null`;
    if (!existsSync(UTMCTL)) die("UTM install failed");
    ok("UTM installed", logFile);
  }

  // Suppress What's New dialog
  const plistPath = "/Applications/UTM.app/Contents/Info.plist";
  if (existsSync(plistPath)) {
    const ver = (await $`/usr/libexec/PlistBuddy -c "Print :CFBundleShortVersionString" ${plistPath}`.quiet().nothrow()).stdout.toString().trim();
    if (ver) {
      const prefs = `${process.env.HOME}/Library/Containers/com.utmapp.UTM/Data/Library/Preferences`;
      mkdirSync(prefs, { recursive: true });
      await $`defaults write ${prefs}/com.utmapp.UTM ReleaseNotesLastVersion -string ${ver}`;
    }
  }

  // Launch and wait
  await $`open -g /Applications/UTM.app`;
  info("Waiting for UTM...", logFile);
  for (let i = 0; i < 30; i++) {
    const r = await $`${UTMCTL} list`.quiet().nothrow();
    if (r.exitCode === 0) {
      ok("UTM ready", logFile);
      return;
    }
    await Bun.sleep(1000);
  }
  die("UTM did not become ready after 30s");
}

// ── Box download + import ────────────────────────────────────────────────────

function parseVmLine(line: string): { uuid: string; status: string; name: string } {
  const parts = line.split(/\s+/);
  return { uuid: parts[0] ?? "", status: parts[1] ?? "", name: parts.slice(2).join(" ").trim() };
}

async function getFirstVm(): Promise<string> {
  const r = await $`${UTMCTL} list`.quiet().nothrow();
  const lines = (r.stdout?.toString() ?? "").split("\n").filter((l) => l.trim() && !l.startsWith("UUID"));
  return lines[0]?.trim() ?? "";
}

/** Download box from Vagrant Cloud, import into UTM. Returns { uuid, displayName }. */
export async function importBox(
  profile: VMProfile,
  vmName: string,
  logFile?: string,
): Promise<{ uuid: string; displayName: string }> {
  const boxCacheDir = `${process.env.HOME}/.cache/utm-dev`;
  const boxArch = "arm64";
  mkdirSync(boxCacheDir, { recursive: true });

  info("Fetching box version...", logFile);
  const versionsApi = `https://api.cloud.hashicorp.com/vagrant/2022-09-30/registry/utm/box/${profile.box}/versions`;
  const versionsRes = await fetch(versionsApi);
  if (!versionsRes.ok) die("Cannot reach Vagrant API");
  const versionsJson = await versionsRes.text();
  const boxVersion = versionsJson.match(/"name":"([^"]*)"/)?.[1];
  if (!boxVersion) die("Cannot parse box version");
  info(`Latest box version: ${boxVersion}`, logFile);

  const boxFile = `${boxCacheDir}/${profile.box}_${boxVersion}_${boxArch}.box`;

  if (existsSync(boxFile)) {
    ok("Box cached (skipping download)", logFile);
  } else {
    await $`rm -f ${boxCacheDir}/${profile.box}_*.box`.nothrow();
    info("Downloading box (~6 GB) — this takes a while...", logFile);
    const downloadApi = `https://api.cloud.hashicorp.com/vagrant/2022-09-30/registry/utm/box/${profile.box}/version/${boxVersion}/provider/utm/architecture/${boxArch}/download`;
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
    ok(`Box downloaded (${boxVersion})`, logFile);
  }

  info("Extracting box...", logFile);
  const tmpdir = (await $`mktemp -d`.quiet()).stdout.toString().trim();
  await $`tar -xf ${boxFile} -C ${tmpdir}`;
  const utmFolder = (await $`find ${tmpdir} -type d -name "*.utm"`.quiet()).stdout.toString().trim().split("\n")[0];
  if (!utmFolder) {
    await $`rm -rf ${tmpdir}`;
    die("No .utm folder in box");
  }

  info("Importing VM into UTM...", logFile);
  const importResult = await $`osascript -e ${"tell application \"UTM\" to import new virtual machine from POSIX file \"" + utmFolder + "\""}`
    .quiet().nothrow();
  await $`rm -rf ${tmpdir}`;
  if (importResult.exitCode !== 0) die("Import failed");

  // Wait for import to register
  info("Waiting for import...", logFile);
  let vmLine = "";
  for (let i = 0; i < 15; i++) {
    vmLine = await getFirstVm();
    if (vmLine) break;
    await Bun.sleep(2000);
  }
  if (!vmLine) die("Import failed — no VM found after 30s");

  const parsed = parseVmLine(vmLine);
  saveState(vmName, parsed.uuid, parsed.name);
  ok(`${vmName} VM imported (${parsed.name})`, logFile);
  return { uuid: parsed.uuid, displayName: parsed.name };
}

// ── Network configuration ────────────────────────────────────────────────────

/** Configure port forwards on the emulated NIC via AppleScript. */
export async function configureNetwork(
  vmUuid: string,
  profile: VMProfile,
  logFile?: string,
): Promise<void> {
  // Find emulated NIC index
  const emulatedResult = await $`osascript -e ${`
    tell application "UTM"
      set vm to virtual machine id "${vmUuid}"
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

  info(`Configuring port forwards on NIC ${emulatedIndex}...`, logFile);

  const script = `
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

  const tmpScript = `/tmp/utm-port-forward-${Date.now()}.scpt`;
  writeFileSync(tmpScript, script);
  try {
    const r = await $`osascript ${tmpScript} ${vmUuid} --index ${emulatedIndex} ${"TcPp,,22,127.0.0.1," + profile.sshPort} --index ${emulatedIndex} ${"TcPp,,3389,127.0.0.1," + profile.rdpPort} --index ${emulatedIndex} ${"TcPp,,5985,127.0.0.1," + profile.winrmPort}`.quiet().nothrow();
    if (r.exitCode !== 0) die("Failed to configure port forwards");
  } finally {
    await $`rm -f ${tmpScript}`.nothrow();
  }

  ok(`Network: SSH:${profile.sshPort} RDP:${profile.rdpPort} WinRM:${profile.winrmPort}`, logFile);
}

// ── VM lifecycle ─────────────────────────────────────────────────────────────

/** Start a VM by display name with retries. */
export async function startVm(displayName: string, logFile?: string): Promise<void> {
  // Check if already running
  const list = await $`${UTMCTL} list`.quiet().nothrow();
  const vmLine = list.stdout.toString().split("\n").find((l) => l.includes(displayName));
  if (vmLine?.includes("started")) {
    ok(`${displayName} already running`, logFile);
    return;
  }

  info(`Starting ${displayName}...`, logFile);
  for (let attempt = 1; attempt <= 3; attempt++) {
    const r = await $`${UTMCTL} start ${displayName}`.quiet().nothrow();
    if (r.exitCode === 0) {
      ok("VM started", logFile);
      return;
    }
    log(`  retry ${attempt}/3...`, logFile);
    await Bun.sleep(5000);
  }
  die(`Failed to start ${displayName} after 3 attempts`);
}

/** Stop a VM if it's running. */
export async function stopVm(displayName: string, logFile?: string): Promise<void> {
  const list = await $`${UTMCTL} list`.quiet().nothrow();
  const vmLine = list.stdout.toString().split("\n").find((l) => l.includes(displayName));
  if (!vmLine?.includes("started")) return;

  info(`Stopping ${displayName}...`, logFile);
  await $`${UTMCTL} stop ${displayName}`.quiet().nothrow();
  await Bun.sleep(5000);
}

/** Wait for Windows to boot (WinRM responding). */
export async function waitForBoot(winrmPort: number, timeoutSec = 300, logFile?: string): Promise<void> {
  info(`Waiting for Windows to boot (up to ${Math.round(timeoutSec / 60)} min)...`, logFile);
  let elapsed = 0;
  while (elapsed < timeoutSec) {
    try {
      await fetch(`http://127.0.0.1:${winrmPort}/wsman`, { signal: AbortSignal.timeout(2000) });
      ok(`Windows ready (${elapsed}s)`, logFile);
      return;
    } catch {}
    await Bun.sleep(5000);
    elapsed += 5;
    if (elapsed % 30 === 0) log(`  still booting... (${elapsed}s)`, logFile);
  }
  die(`Timeout waiting for Windows (${timeoutSec}s)`);
}
