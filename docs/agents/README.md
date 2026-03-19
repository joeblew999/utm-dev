# AI Agent Collaboration Guide

This directory contains documentation for AI assistants (like Claude Code, GitHub Copilot, Cursor, etc.) to collaborate effectively on the utm-dev project.

## Purpose

AI agents need context about dependencies, architecture, and patterns to work effectively. This directory provides:

1. **Source references** - Links to dependency source code
2. **Architecture guides** - System design and patterns
3. **Collaboration patterns** - How multiple agents can work together

## Available Guides

- [gio-plugins.md](gio-plugins.md) - Guide to the gio-plugins dependency

## Contributing New Agent Docs

When adding a new major dependency or subsystem:

1. Clone the source to `.src/` for easy reference
2. Create a guide in `docs/agents/[dependency-name].md`
3. Document key files, patterns, and integration points
4. Update this README with a link

## For AI Assistants

When working on utm-dev:

1. Check `.src/` for dependency source code
2. Read relevant agent guides before making changes
3. Update guides when discovering new patterns
4. Keep documentation focused and actionable
