#!/usr/bin/env bun

//MISE description="Bootstrap Windows VM: SSH + dev tools (build) or SSH only (test)"
//MISE alias="vm-bootstrap"

// Bootstraps a Windows VM via WinRM (the only thing available on a fresh box).
// - "full" (build VM): OpenSSH + VS Build Tools + WebView2 + mise
// - "ssh-only" (test VM): just OpenSSH — clean Windows for testing
// Idempotent — safe to run multiple times.
// Usage: vm:bootstrap [build|test]  (default: build)

import { parseVMArg, getProfile, info, ok, die, log, timestamp } from "../_lib.ts";
import { WinRM } from "../_winrm.ts";

const LOG = "vm-bootstrap.log";
log(`── ${timestamp()} ──`, LOG);

const { vmName } = parseVMArg();
const profile = getProfile(vmName);

const winrm = new WinRM("127.0.0.1", profile.winrmPort, profile.user, profile.pass);

// ── Check WinRM is reachable ──────────────────────────────────────────────

if (!(await winrm.ping())) {
  die(`WinRM not reachable on port ${profile.winrmPort} — is the ${vmName} VM running?`);
}

info(`Bootstrapping ${vmName} VM via WinRM (mode: ${profile.bootstrap})...`, LOG);

// ── Helpers ───────────────────────────────────────────────────────────────

async function check(desc: string, psCheck: string): Promise<boolean> {
  const r = await winrm.runPS(psCheck);
  const val = r.stdout.trim();
  if (val) {
    ok(`${desc}: ${val}`, LOG);
    return true;
  }
  return false;
}

async function wingetInstall(pkgId: string, desc: string, timeout = 300): Promise<void> {
  if (await check(desc, `winget list --id ${pkgId} --accept-source-agreements 2>$null | Select-String "${pkgId}"`)) {
    return;
  }
  info(`Installing ${desc} via winget...`, LOG);
  await winrm.runElevated(
    `winget install --id ${pkgId} --accept-source-agreements --accept-package-agreements --silent`,
    timeout,
  );
}

// ── Step 1: OpenSSH Server (all modes) ──────────────────────────────────

if (!(await check("OpenSSH", "Get-Service sshd -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Status"))) {
  info("Installing OpenSSH Server...", LOG);
  await winrm.runElevated("Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0", 300);
}

info("Configuring SSH...", LOG);
await winrm.runElevated(`$configPath = "C:\\ProgramData\\ssh\\sshd_config"
$content = Get-Content $configPath
$newContent = @()
foreach ($line in $content) {
    if ($line -match "^#PasswordAuthentication yes") {
        $newContent += "PasswordAuthentication yes"
    } elseif ($line -match "^Match Group administrators") {
        $newContent += "#Match Group administrators"
    } elseif ($line -match "AuthorizedKeysFile __PROGRAMDATA__") {
        $newContent += "#AuthorizedKeysFile __PROGRAMDATA__/ssh/administrators_authorized_keys"
    } else {
        $newContent += $line
    }
}
$newContent | Set-Content $configPath -Force
Start-Service sshd -ErrorAction SilentlyContinue
Set-Service -Name sshd -StartupType Automatic
Restart-Service sshd`);

const sshdStatus = await winrm.runPS("Get-Service sshd | Select-Object -ExpandProperty Status");
if (sshdStatus.stdout.trim() !== "Running") {
  die(`sshd: ${sshdStatus.stdout.trim()}`);
}
ok(`sshd: ${sshdStatus.stdout.trim()}`, LOG);

// ── ssh-only mode stops here ────────────────────────────────────────────

if (profile.bootstrap === "ssh-only") {
  log("", LOG);
  ok(`${vmName} VM bootstrap complete (SSH only)`, LOG);
  log(`  SSH: sshpass -p ${profile.pass} ssh -p ${profile.sshPort} ${profile.user}@127.0.0.1`, LOG);
  log(`  RDP: localhost:${profile.rdpPort}`, LOG);
  process.exit(0);
}

// ── Full mode: dev tools ────────────────────────────────────────────────

// Step 2: VS Build Tools (needed for Rust/MSVC on Windows)
await wingetInstall("Microsoft.VisualStudio.2022.BuildTools", "VS Build Tools 2022", 600);

info("Installing C++ workload for MSVC...", LOG);
await winrm.runElevated(`$vs = & "C:\\Program Files (x86)\\Microsoft Visual Studio\\Installer\\vswhere.exe" -latest -property installationPath 2>$null
if (-not $vs) { $vs = "C:\\Program Files\\Microsoft Visual Studio\\2022\\BuildTools" }
$installer = "C:\\Program Files (x86)\\Microsoft Visual Studio\\Installer\\vs_installer.exe"
if (Test-Path $installer) {
    Start-Process -FilePath $installer -ArgumentList "modify","--installPath",$vs,"--add","Microsoft.VisualStudio.Workload.VCTools","--includeRecommended","--quiet","--norestart" -Wait -NoNewWindow
}`, 600);

// Step 3: WebView2 Runtime (needed by Tauri)
await wingetInstall("Microsoft.EdgeWebView2Runtime", "WebView2 Runtime", 120);

// Step 4: mise (handles Rust + cargo-tauri)
const miseCheck = await winrm.runCmd("where mise");
if (miseCheck.exitCode === 0) {
  const ver = await winrm.runCmd("mise --version");
  ok(`mise: ${ver.stdout.trim()}`, LOG);
} else {
  info("Installing mise...", LOG);
  const r = await winrm.runPS("winget install --id jdx.mise --accept-source-agreements --accept-package-agreements --silent");
  if (r.exitCode === 0) {
    ok("mise installed", LOG);
  } else {
    const r2 = await winrm.runPS('Invoke-Expression (Invoke-WebRequest -Uri "https://mise.run" -UseBasicParsing).Content');
    log(`  mise install: exit ${r2.exitCode}`, LOG);
  }
}

log("", LOG);
ok(`${vmName} VM bootstrap complete`, LOG);
log("  Next: mise run vm:sync && mise run vm:exec 'cd project && mise install'", LOG);
