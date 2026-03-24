#!/usr/bin/env bun

//MISE description="Set up MCP servers for AI-assisted Tauri development"
//MISE alias="m"

// Writes .mcp.json (MCP server config) and .claude/settings.json (auto-allow
// permissions) to the current project root. Idempotent — safe to run repeatedly.

import { existsSync, mkdirSync, readFileSync, writeFileSync } from "fs";
import { execSync } from "child_process";
import { join } from "path";
import { info, ok, die } from "./_lib.ts";

const root = process.cwd();
const mcpJson = join(root, ".mcp.json");
const claudeDir = join(root, ".claude");
const claudeSettings = join(claudeDir, "settings.json");

type McpConfig = {
  mcpServers: Record<string, { command: string; args: string[]; env?: Record<string, string> }>;
};

// ── Resolve full paths (Claude Code can't find binaries via PATH) ────────────

function resolveBin(name: string, miseTool?: string): string {
  if (miseTool) {
    try {
      const toolDir = execSync(`mise where ${miseTool}`, { encoding: "utf-8" }).trim();
      const fullPath = join(toolDir, "bin", name);
      if (existsSync(fullPath)) return fullPath;
    } catch {}
  }
  // fallback: check if binary is on PATH
  try {
    return execSync(`which ${name}`, { encoding: "utf-8" }).trim();
  } catch {}
  return name; // bare fallback — may not work in sandboxed envs
}

const bunx = resolveBin("bunx", "bun");
const mise = resolveBin("mise");

// ── Desired servers ──────────────────────────────────────────────────────────

const SERVERS: McpConfig["mcpServers"] = {
  context7: {
    command: bunx,
    args: ["@upstash/context7-mcp@latest"],
  },
  mise: {
    command: mise,
    args: ["mcp"],
    env: { MISE_EXPERIMENTAL: "true" },
  },
};

// ── MCP tool permissions for Claude Code ─────────────────────────────────────

// Auto-allow all tools from configured MCP servers so Claude Code doesn't
// prompt the user for permission on every call.
function mcpPermissions(): string[] {
  const perms: string[] = [];
  for (const name of Object.keys(SERVERS)) {
    perms.push(`mcp__${name}__*`);
  }
  return perms;
}

// ── 1. Write .mcp.json ──────────────────────────────────────────────────────

let config: McpConfig = { mcpServers: {} };

if (existsSync(mcpJson)) {
  try {
    config = JSON.parse(readFileSync(mcpJson, "utf-8"));
    config.mcpServers ??= {};
  } catch {
    die(`${mcpJson} exists but is not valid JSON`);
  }
}

let mcpAdded = 0;
for (const [name, server] of Object.entries(SERVERS)) {
  if (config.mcpServers[name]) {
    ok(`${name} (already configured)`);
  } else {
    config.mcpServers[name] = server;
    info(`Adding ${name}`);
    mcpAdded++;
  }
}

if (mcpAdded > 0) {
  writeFileSync(mcpJson, JSON.stringify(config, null, 2) + "\n");
  ok(`Wrote ${mcpJson}`);
}

// ── 2. Write .claude/settings.json (permissions) ────────────────────────────

type ClaudeSettings = {
  permissions?: {
    allow?: string[];
    [key: string]: unknown;
  };
  [key: string]: unknown;
};

let claude: ClaudeSettings = {};

if (existsSync(claudeSettings)) {
  try {
    claude = JSON.parse(readFileSync(claudeSettings, "utf-8"));
  } catch {
    die(`${claudeSettings} exists but is not valid JSON`);
  }
}

claude.permissions ??= {};
claude.permissions.allow ??= [];

const needed = mcpPermissions();
let permsAdded = 0;
for (const perm of needed) {
  if (!claude.permissions.allow.includes(perm)) {
    claude.permissions.allow.push(perm);
    info(`Allowing ${perm}`);
    permsAdded++;
  }
}

if (permsAdded > 0) {
  mkdirSync(claudeDir, { recursive: true });
  writeFileSync(claudeSettings, JSON.stringify(claude, null, 2) + "\n");
  ok(`Wrote ${claudeSettings}`);
} else {
  ok("Permissions already configured");
}

if (mcpAdded === 0 && permsAdded === 0) {
  ok("Nothing to do");
}
