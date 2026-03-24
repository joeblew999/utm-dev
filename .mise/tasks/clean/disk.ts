#!/usr/bin/env bun

//MISE description="Free system-wide disk space (all Rust targets, Xcode, Gradle, Bun caches)"
//MISE alias="clean"

import { $ } from "bun";
import { existsSync } from "fs";
import { log } from "../_lib.ts";

const HOME = process.env.HOME!;
const DRY = process.argv.includes("--dry-run");
const DEEP = process.argv.includes("--deep");

// ── Types ──────────────────────────────────────────────────────────────────

interface Target {
  label: string;
  path: string;
  bytes: number;
  human: string;
  clean: () => Promise<void>;
}

const PROTECTED = [
  { path: `${HOME}/.cache/utm-dev`, reason: "box images (~6 GB download)" },
  { path: `${HOME}/Library/Containers/com.utmapp.UTM`, reason: "your VMs" },
  { path: `${HOME}/.rustup/toolchains`, reason: "Rust toolchains" },
  { path: `${HOME}/.android-sdk`, reason: "Android SDK" },
];

// ── Helpers ────────────────────────────────────────────────────────────────

async function dirBytes(path: string): Promise<number> {
  if (!existsSync(path)) return 0;
  const r = await $`du -sk ${path}`.quiet().nothrow();
  if (r.exitCode !== 0) return 0;
  return parseInt(r.stdout.toString().split("\t")[0]) * 1024;
}

function fmt(bytes: number): string {
  if (bytes >= 1_073_741_824) return `${(bytes / 1_073_741_824).toFixed(1)} GB`;
  if (bytes >= 1_048_576) return `${(bytes / 1_048_576).toFixed(0)} MB`;
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  return `${bytes} B`;
}

async function diskFree(): Promise<{ avail: string; used: string; total: string; pct: string }> {
  const r = await $`df -h /System/Volumes/Data`.quiet().nothrow();
  const parts = r.stdout.toString().trim().split("\n")[1]?.split(/\s+/) ?? [];
  return { total: parts[1] ?? "?", used: parts[2] ?? "?", avail: parts[3] ?? "?", pct: parts[4] ?? "?" };
}

// ── Scan ───────────────────────────────────────────────────────────────────

async function scan(): Promise<Target[]> {
  const targets: Target[] = [];

  // 1. Rust target/ directories
  const searchDirs = [`${HOME}/workspace`, `${HOME}/src`, `${HOME}/projects`, `${HOME}/code`];
  for (const dir of searchDirs) {
    if (!existsSync(dir)) continue;
    const r = await $`find ${dir} -name target -type d -maxdepth 6 2>/dev/null`.quiet().nothrow();
    for (const line of r.stdout.toString().trim().split("\n").filter(Boolean)) {
      const isRust = existsSync(`${line}/debug`) || existsSync(`${line}/release`) || existsSync(`${line}/.rustc_info.json`);
      if (!isRust) continue;
      const bytes = await dirBytes(line);
      if (bytes < 50_000_000) continue; // skip tiny targets (<50 MB)
      const project = line.replace(/\/target$/, "").split("/").slice(-2).join("/");
      const cargoToml = line.replace(/\/target$/, "/Cargo.toml");
      targets.push({
        label: `Rust: ${project}`,
        path: line,
        bytes,
        human: fmt(bytes),
        clean: async () => {
          if (existsSync(cargoToml)) {
            await $`cargo clean --manifest-path ${cargoToml}`.quiet().nothrow();
          } else {
            await $`rm -rf ${line}`.quiet().nothrow();
          }
        },
      });
    }
  }

  // 2. iOS simulators (unavailable)
  const simUnavail = await $`xcrun simctl list devices unavailable -j 2>/dev/null`.quiet().nothrow();
  if (simUnavail.exitCode === 0) {
    try {
      const data = JSON.parse(simUnavail.stdout.toString());
      let count = 0;
      for (const devices of Object.values(data.devices) as any[][]) count += devices.length;
      if (count > 0) {
        // Estimate ~500 MB per unavailable simulator
        const est = count * 500_000_000;
        targets.push({
          label: `iOS simulators (${count} unavailable)`,
          path: "xcrun simctl delete unavailable",
          bytes: est,
          human: `~${fmt(est)}`,
          clean: async () => { await $`xcrun simctl delete unavailable`.quiet().nothrow(); },
        });
      }
    } catch {}
  }

  // 3. CoreSimulator caches & logs (not the devices themselves)
  const simCaches = `${HOME}/Library/Developer/CoreSimulator/Caches`;
  const simCacheBytes = await dirBytes(simCaches);
  if (simCacheBytes > 10_000_000) {
    targets.push({
      label: "CoreSimulator caches",
      path: simCaches,
      bytes: simCacheBytes,
      human: fmt(simCacheBytes),
      clean: async () => { await $`rm -rf ${simCaches}`.quiet().nothrow(); },
    });
  }

  // 4. Xcode derived data
  const dd = `${HOME}/Library/Developer/Xcode/DerivedData`;
  const ddBytes = await dirBytes(dd);
  if (ddBytes > 10_000_000) {
    targets.push({
      label: "Xcode DerivedData",
      path: dd,
      bytes: ddBytes,
      human: fmt(ddBytes),
      clean: async () => { await $`rm -rf ${dd}`.quiet().nothrow(); },
    });
  }

  // 5. Cargo registry cache
  const cargoReg = `${HOME}/.cargo/registry/cache`;
  const cargoRegBytes = await dirBytes(cargoReg);
  if (cargoRegBytes > 10_000_000) {
    targets.push({
      label: "Cargo registry cache",
      path: cargoReg,
      bytes: cargoRegBytes,
      human: fmt(cargoRegBytes),
      clean: async () => { await $`rm -rf ${cargoReg}`.quiet().nothrow(); },
    });
  }

  // 6. Gradle caches
  const gradle = `${HOME}/.gradle/caches`;
  const gradleBytes = await dirBytes(gradle);
  if (gradleBytes > 10_000_000) {
    targets.push({
      label: "Gradle caches",
      path: gradle,
      bytes: gradleBytes,
      human: fmt(gradleBytes),
      clean: async () => { await $`rm -rf ${gradle}`.quiet().nothrow(); },
    });
  }

  // 7. Bun install cache
  const bunCache = `${HOME}/.bun/install/cache`;
  const bunBytes = await dirBytes(bunCache);
  if (bunBytes > 10_000_000) {
    targets.push({
      label: "Bun install cache",
      path: bunCache,
      bytes: bunBytes,
      human: fmt(bunBytes),
      clean: async () => { await $`rm -rf ${bunCache}`.quiet().nothrow(); },
    });
  }

  // 8. npm cache
  const npmCache = `${HOME}/.npm/_cacache`;
  const npmBytes = await dirBytes(npmCache);
  if (npmBytes > 10_000_000) {
    targets.push({
      label: "npm cache",
      path: npmCache,
      bytes: npmBytes,
      human: fmt(npmBytes),
      clean: async () => { await $`npm cache clean --force`.quiet().nothrow(); },
    });
  }

  // 9. CocoaPods cache
  const pods = `${HOME}/Library/Caches/CocoaPods`;
  const podsBytes = await dirBytes(pods);
  if (podsBytes > 10_000_000) {
    targets.push({
      label: "CocoaPods cache",
      path: pods,
      bytes: podsBytes,
      human: fmt(podsBytes),
      clean: async () => { await $`rm -rf ${pods}`.quiet().nothrow(); },
    });
  }

  // ── Deep clean extras (only with --deep) ────────────────────────────────
  if (DEEP) {
    // 10. Homebrew cache
    const brewCache = `${HOME}/Library/Caches/Homebrew`;
    const brewBytes = await dirBytes(brewCache);
    if (brewBytes > 10_000_000) {
      targets.push({
        label: "Homebrew cache",
        path: brewCache,
        bytes: brewBytes,
        human: fmt(brewBytes),
        clean: async () => { await $`brew cleanup --prune=all`.quiet().nothrow(); },
      });
    }

    // 11. Xcode archives
    const archives = `${HOME}/Library/Developer/Xcode/Archives`;
    const archBytes = await dirBytes(archives);
    if (archBytes > 10_000_000) {
      targets.push({
        label: "Xcode Archives",
        path: archives,
        bytes: archBytes,
        human: fmt(archBytes),
        clean: async () => { await $`rm -rf ${archives}`.quiet().nothrow(); },
      });
    }

    // 12. Xcode device support
    const devSupport = `${HOME}/Library/Developer/Xcode/iOS DeviceSupport`;
    const devBytes = await dirBytes(devSupport);
    if (devBytes > 10_000_000) {
      targets.push({
        label: "Xcode iOS DeviceSupport",
        path: devSupport,
        bytes: devBytes,
        human: fmt(devBytes),
        clean: async () => { await $`rm -rf ${devSupport}`.quiet().nothrow(); },
      });
    }

    // 13. Docker (if present)
    const dockerSock = "/var/run/docker.sock";
    if (existsSync(dockerSock)) {
      const r = await $`docker system df --format '{{.Size}}' 2>/dev/null`.quiet().nothrow();
      if (r.exitCode === 0 && r.stdout.toString().trim()) {
        targets.push({
          label: "Docker (unused images, build cache)",
          path: "docker system prune",
          bytes: 0, // can't easily estimate
          human: "varies",
          clean: async () => { await $`docker system prune -af`.quiet().nothrow(); },
        });
      }
    }

    // 14. macOS system logs
    const sysLogs = "/private/var/log";
    const logBytes = await dirBytes(sysLogs);
    if (logBytes > 500_000_000) {
      targets.push({
        label: "System logs (sudo)",
        path: sysLogs,
        bytes: logBytes,
        human: fmt(logBytes),
        clean: async () => { await $`sudo rm -rf /private/var/log/asl/*.asl`.quiet().nothrow(); },
      });
    }
  }

  // Sort biggest first
  targets.sort((a, b) => b.bytes - a.bytes);
  return targets;
}

// ── Main ───────────────────────────────────────────────────────────────────

const before = await diskFree();
log(`\n═══ utm-dev clean ═══`);
log(`Disk: ${before.avail} free of ${before.total} (${before.pct} used)`);
if (DRY) log(`Mode: dry run`);
if (DEEP) log(`Mode: deep clean`);
log("");

log("Scanning...");
const targets = await scan();

if (targets.length === 0) {
  log("\nNothing to clean (everything under 50 MB threshold).");
  log("Try --deep for more aggressive cleanup.\n");
  process.exit(0);
}

// ── Display what we found ──────────────────────────────────────────────────
const totalBytes = targets.reduce((sum, t) => sum + t.bytes, 0);
const pad = (s: string, n: number) => s.padEnd(n);

log("");
log(`  #  ${"What".padEnd(40)} Size`);
log(`  ${"─".repeat(52)}`);
for (let i = 0; i < targets.length; i++) {
  const t = targets[i];
  log(`  ${String(i + 1).padStart(2)}  ${pad(t.label, 40)} ${t.human}`);
}
log(`  ${"─".repeat(52)}`);
log(`      ${"TOTAL".padEnd(40)} ~${fmt(totalBytes)}`);
log("");

if (DRY) {
  log("Dry run — nothing was deleted. Remove --dry-run to clean.");
  if (!DEEP) log("Add --deep for Homebrew, Xcode archives, Docker, device support.");
  log("");
  process.exit(0);
}

// ── Clean ──────────────────────────────────────────────────────────────────
log("Cleaning...\n");
let cleaned = 0;
for (const t of targets) {
  process.stdout.write(`  ${t.label}...`);
  try {
    await t.clean();
    log(` ${t.human} freed`);
    cleaned += t.bytes;
  } catch (e) {
    log(` FAILED (${e})`);
  }
}

// ── Summary ────────────────────────────────────────────────────────────────
const after = await diskFree();
log(`\n═══ Done ═══`);
log(`Freed: ~${fmt(cleaned)}`);
log(`Disk:  ${before.avail} -> ${after.avail} free (${after.pct} used)`);

log(`\nProtected (never touched):`);
for (const p of PROTECTED) {
  const exists = existsSync(p.path);
  const short = p.path.replace(HOME, "~");
  if (exists) {
    const bytes = await dirBytes(p.path);
    log(`  ${short} (${fmt(bytes)}) — ${p.reason}`);
  }
}

if (!DEEP) {
  log(`\nTip: use --deep for Homebrew, Xcode archives, Docker, device support.`);
}
log("");
