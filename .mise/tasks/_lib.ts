// Shared constants and helpers for vm:* Bun tasks.
// Import: import { ssh, scp, info, ok, die, ... } from "./_lib.ts";

import { mkdirSync, appendFileSync, existsSync, readFileSync } from "fs";
import { join } from "path";
import { $ } from "bun";

// ── VM constants ──────────────────────────────────────────────────────────

export const SSH_PORT = 2222;
export const RDP_PORT = 3389;
export const WINRM_PORT = 5985;
export const VM_USER = "vagrant";
export const VM_PASS = "vagrant";
export const VM_HOME = `C:\\Users\\${VM_USER}`;
export const UTMCTL = "/Applications/UTM.app/Contents/MacOS/utmctl";

export const PROJECT_DIR = process.cwd();
export const PROJECT_NAME = PROJECT_DIR.split("/").pop()!;
export const LOGDIR = join(PROJECT_DIR, ".mise", "logs");
export const STATEFILE = join(PROJECT_DIR, ".mise", "state", "vm.env");

mkdirSync(LOGDIR, { recursive: true });

// ── Logging ───────────────────────────────────────────────────────────────

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

// ── SSH helpers ───────────────────────────────────────────────────────────

export async function ensureSshpass(): Promise<void> {
  const which = await $`command -v sshpass`.quiet().nothrow();
  if (which.exitCode === 0) return;
  info("Installing sshpass...");
  await $`HOMEBREW_NO_AUTO_UPDATE=1 brew install hudochenkov/sshpass/sshpass < /dev/null 2>/dev/null`;
}

export async function ssh(
  cmd: string,
  opts: { quiet?: boolean } = {},
): Promise<{ stdout: string; stderr: string; exitCode: number }> {
  const result = await $`sshpass -p ${VM_PASS} ssh -o StrictHostKeyChecking=no -p ${SSH_PORT} ${VM_USER}@127.0.0.1 ${cmd}`
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
  src: string,
  dst: string,
): Promise<void> {
  await $`sshpass -p ${VM_PASS} scp -o StrictHostKeyChecking=no -P ${SSH_PORT} ${src} ${dst}`;
}

export async function checkSsh(): Promise<void> {
  const result = await ssh("echo ok", { quiet: true });
  if (!result.stdout.includes("ok")) {
    die("Cannot connect via SSH. Run: mise run vm:up");
  }
}

// ── State file helpers ────────────────────────────────────────────────────

export function loadState(): { VM_UUID: string; VM_DISPLAY_NAME: string } {
  if (!existsSync(STATEFILE)) {
    die("No VM state — run mise run vm:up first");
  }
  const content = readFileSync(STATEFILE, "utf-8");
  const uuid = content.match(/VM_UUID="([^"]*)"/)?.[1] ?? "";
  const name = content.match(/VM_DISPLAY_NAME="([^"]*)"/)?.[1] ?? "";
  if (!uuid) die("No VM UUID in state");
  return { VM_UUID: uuid, VM_DISPLAY_NAME: name };
}
