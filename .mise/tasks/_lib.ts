// Shared constants and helpers for vm:* Bun tasks.
// Import: import { getProfile, ssh, scp, info, ok, die, ... } from "../_lib.ts";

import { mkdirSync, appendFileSync, existsSync, readFileSync, writeFileSync, renameSync, unlinkSync } from "fs";
import { join } from "path";
import { $ } from "bun";

// ── VM profiles ──────────────────────────────────────────────────────────────

export type GuestOS = "windows" | "linux";
export type BootstrapMode = "full" | "ssh-only" | false;

export type VMProfile = {
  os: GuestOS;
  box: string;
  sshPort: number;
  rdpPort?: number;     // Windows only
  winrmPort?: number;   // Windows only
  user: string;
  pass: string;
  bootstrap: BootstrapMode;
  memoryMiB: number;    // RAM in MiB
  cpuCores: number;     // CPU cores
};

export const VM_PROFILES: Record<string, VMProfile> = {
  "windows-build": {
    os: "windows",
    box: "windows-11",
    sshPort: 2222,
    rdpPort: 3389,
    winrmPort: 5985,
    user: "vagrant",
    pass: "vagrant",
    bootstrap: "full",      // SSH + VS Build Tools + WebView2 + mise
    memoryMiB: 12288,       // 12 GB — VS Build Tools install crashes VM at 8 GB on ARM64
    cpuCores: 4,
  },
  "windows-test": {
    os: "windows",
    box: "windows-11",
    sshPort: 2322,
    rdpPort: 3489,
    winrmPort: 6985,
    user: "vagrant",
    pass: "vagrant",
    bootstrap: "ssh-only",  // SSH only — clean Windows for testing
    memoryMiB: 4096,        // 4 GB — just running the app
    cpuCores: 2,
  },
  "linux-build": {
    os: "linux",
    box: "ubuntu-24.04",
    sshPort: 2422,
    user: "vagrant",
    pass: "vagrant",
    bootstrap: "full",      // build-essential + Rust + mise + Tauri deps
    memoryMiB: 4096,        // 4 GB — Linux + Rust is lighter
    cpuCores: 4,
  },
  "linux-test": {
    os: "linux",
    box: "ubuntu-24.04",
    sshPort: 2522,
    user: "vagrant",
    pass: "vagrant",
    bootstrap: "ssh-only",  // SSH only — clean Linux for testing
    memoryMiB: 2048,        // 2 GB — just running the app
    cpuCores: 2,
  },
};

export const DEFAULT_VM = "windows-build";

export function getProfile(name?: string): VMProfile & { name: string } {
  const vmName = name || DEFAULT_VM;
  const profile = VM_PROFILES[vmName];
  if (!profile) {
    die(`Unknown VM profile: ${vmName}. Available: ${Object.keys(VM_PROFILES).join(", ")}`);
  }
  return { ...profile, name: vmName };
}

/** Parse VM name from process.argv. Returns the name and remaining args. */
export function parseVMArg(argv = process.argv): { vmName: string; rest: string[] } {
  // argv: [bun, script, ...args]
  const args = argv.slice(2);
  if (args.length > 0 && args[0] in VM_PROFILES) {
    return { vmName: args[0], rest: args.slice(1) };
  }
  return { vmName: DEFAULT_VM, rest: args };
}

// ── Global constants ─────────────────────────────────────────────────────────

export const UTMCTL = "/Applications/UTM.app/Contents/MacOS/utmctl";

export const PROJECT_DIR = process.cwd();
export const PROJECT_NAME = PROJECT_DIR.split("/").pop()!;
export const LOGDIR = join(PROJECT_DIR, ".mise", "logs");
export const STATE_DIR = join(PROJECT_DIR, ".mise", "state");

mkdirSync(LOGDIR, { recursive: true });
mkdirSync(STATE_DIR, { recursive: true });

// ── Logging ──────────────────────────────────────────────────────────────────

export function log(msg: string, logFile?: string) {
  const line = msg + "\n";
  process.stdout.write(line);
  if (logFile) {
    appendFileSync(join(LOGDIR, logFile), line);
  }
}

export function info(msg: string, logFile?: string) {
  log(`→ ${msg}`, logFile);
}

export function ok(msg: string, logFile?: string) {
  log(`✓ ${msg}`, logFile);
}

export function die(msg: string): never {
  process.stderr.write(`✗ ${msg}\n`);
  process.exit(1);
}

export function timestamp(): string {
  return new Date().toISOString().replace("T", " ").slice(0, 19);
}

// ── SSH helpers (profile-aware) ──────────────────────────────────────────────

export async function ensureSshpass(): Promise<void> {
  const which = await $`command -v sshpass`.quiet().nothrow();
  if (which.exitCode === 0) return;
  info("Installing sshpass...");
  await $`HOMEBREW_NO_AUTO_UPDATE=1 brew install hudochenkov/sshpass/sshpass < /dev/null 2>/dev/null`;
}

export async function ssh(
  profile: VMProfile,
  cmd: string,
  opts: { quiet?: boolean } = {},
): Promise<{ stdout: string; stderr: string; exitCode: number }> {
  const result = await $`sshpass -p ${profile.pass} ssh -o StrictHostKeyChecking=no -p ${profile.sshPort} ${profile.user}@127.0.0.1 ${cmd}`
    .quiet()
    .nothrow();
  if (!opts.quiet) {
    if (result.stdout.length) process.stdout.write(result.stdout);
    if (result.stderr.length) process.stderr.write(result.stderr);
  }
  return {
    stdout: result.stdout.toString().trim(),
    stderr: result.stderr.toString().trim(),
    exitCode: result.exitCode,
  };
}

export async function scp(
  profile: VMProfile,
  src: string,
  dst: string,
): Promise<void> {
  await $`sshpass -p ${profile.pass} scp -o StrictHostKeyChecking=no -P ${profile.sshPort} ${src} ${dst}`;
}

export async function checkSsh(profile: VMProfile): Promise<void> {
  const result = await ssh(profile, "echo ok", { quiet: true });
  if (!result.stdout.includes("ok")) {
    die(`Cannot connect via SSH on port ${profile.sshPort}. Run: mise run vm:up ${(profile as any).name || ""}`);
  }
}

// ── State file helpers (per-VM) ──────────────────────────────────────────────

function stateFile(vmName: string): string {
  return join(STATE_DIR, `vm-${vmName}.env`);
}

export function loadState(vmName: string): { VM_UUID: string; VM_DISPLAY_NAME: string } {
  // Migrate old single state file on first access
  const oldFile = join(STATE_DIR, "vm.env");
  const newFile = stateFile(vmName);
  if (!existsSync(newFile) && existsSync(oldFile) && vmName === DEFAULT_VM) {
    renameSync(oldFile, newFile);
  }

  if (!existsSync(newFile)) {
    die(`No VM state for "${vmName}" — run: mise run vm:up ${vmName}`);
  }
  const content = readFileSync(newFile, "utf-8");
  const uuid = content.match(/VM_UUID="([^"]*)"/)?.[1] ?? "";
  const name = content.match(/VM_DISPLAY_NAME="([^"]*)"/)?.[1] ?? "";
  if (!uuid) die(`No VM UUID in state for "${vmName}"`);
  return { VM_UUID: uuid, VM_DISPLAY_NAME: name };
}

export function saveState(vmName: string, uuid: string, displayName: string): void {
  writeFileSync(stateFile(vmName), `VM_UUID="${uuid}"\nVM_DISPLAY_NAME="${displayName}"\n`);
}

export function hasState(vmName: string): boolean {
  return existsSync(stateFile(vmName));
}

export function clearState(vmName: string): void {
  const f = stateFile(vmName);
  if (existsSync(f)) {
    unlinkSync(f);
  }
}

// ── VM Home path helper ─────────────────────────────────────────────────────

export function vmHome(profile: VMProfile): string {
  if (profile.os === "linux") return `/home/${profile.user}`;
  return `C:\\Users\\${profile.user}`;
}
