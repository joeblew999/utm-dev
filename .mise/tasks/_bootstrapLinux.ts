// Internal — called by vm/up.ts for Linux VMs, not a user-facing task.

// Bootstraps a Linux VM via SSH (already available on Vagrant boxes).
// - "full" (linux-build VM): build-essential + Rust + mise + Tauri Linux deps
// - "ssh-only" (linux-test VM): verify SSH works, done
// Idempotent — safe to run multiple times.

import { parseVMArg, getProfile, ensureSshpass, ssh, info, ok, die, log, timestamp } from "./_lib.ts";

const LOG = "vm-bootstrap-linux.log";
log(`── ${timestamp()} ──`, LOG);

const { vmName } = parseVMArg();
const profile = getProfile(vmName);

await ensureSshpass();

// ── Check SSH is reachable ──────────────────────────────────────────────

const ping = await ssh(profile, "echo ok", { quiet: true });
if (!ping.stdout.includes("ok")) {
  die(`SSH not reachable on port ${profile.sshPort} — is the ${vmName} VM running?`);
}

info(`Bootstrapping ${vmName} VM via SSH (mode: ${profile.bootstrap})...`, LOG);

// ── Helpers ───────────────────────────────────────────────────────────────

async function run(desc: string, cmd: string): Promise<void> {
  info(`${desc}...`, LOG);
  const r = await ssh(profile, cmd);
  if (r.exitCode !== 0) {
    log(`  ⚠ ${desc} may have failed (exit ${r.exitCode})`, LOG);
  }
}

async function check(desc: string, cmd: string): Promise<boolean> {
  const r = await ssh(profile, cmd, { quiet: true });
  const val = r.stdout.trim();
  if (val && r.exitCode === 0) {
    ok(`${desc}: ${val}`, LOG);
    return true;
  }
  return false;
}

// ── ssh-only mode stops here ────────────────────────────────────────────

if (profile.bootstrap === "ssh-only") {
  ok(`${vmName} VM bootstrap complete (SSH verified)`, LOG);
  log(`  SSH: sshpass -p ${profile.pass} ssh -p ${profile.sshPort} ${profile.user}@127.0.0.1`, LOG);
  process.exit(0);
}

// ── Full mode: dev tools ────────────────────────────────────────────────

// Step 1: Update package list + install build essentials
if (!(await check("build-essential", "dpkg -s build-essential 2>/dev/null | grep -oP 'ok installed'"))) {
  await run("Updating apt and installing build-essential", "sudo DEBIAN_FRONTEND=noninteractive apt-get update -qq && sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq build-essential curl git pkg-config");
}

// Step 2: Tauri Linux dependencies (WebKitGTK + GTK + system libs)
// Matches official Tauri prerequisites: https://v2.tauri.app/start/prerequisites/
const tauriDeps = [
  "libwebkit2gtk-4.1-dev",
  "libgtk-3-dev",
  "libayatana-appindicator3-dev",
  "librsvg2-dev",
  "libssl-dev",
  "libxdo-dev",             // required by tauri-plugin-global-shortcut
  "patchelf",
  "wget",
  "file",
  "libsoup-3.0-dev",
  "libjavascriptcoregtk-4.1-dev",
].join(" ");

if (!(await check("libwebkit2gtk-4.1-dev", "dpkg -s libwebkit2gtk-4.1-dev 2>/dev/null | grep -oP 'ok installed'"))) {
  await run("Installing Tauri Linux dependencies", `sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq ${tauriDeps}`);
}

// Step 3: Rust (via rustup)
// Check with full path since ~/.cargo/bin may not be on PATH in this SSH session
if (!(await check("Rust", "~/.cargo/bin/rustc --version 2>/dev/null || rustc --version 2>/dev/null"))) {
  await run("Installing Rust via rustup", 'curl --proto "=https" --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y');
}

// Step 4: mise
// Check with full path since ~/.local/bin may not be on PATH in this SSH session
if (!(await check("mise", "~/.local/bin/mise --version 2>/dev/null || mise --version 2>/dev/null"))) {
  await run("Installing mise", 'curl https://mise.run | sh');
}

log("", LOG);
ok(`${vmName} VM bootstrap complete`, LOG);
log(`  SSH: sshpass -p ${profile.pass} ssh -p ${profile.sshPort} ${profile.user}@127.0.0.1`, LOG);
log("  Next: mise run vm:exec linux-build 'cd <project> && mise install'", LOG);
