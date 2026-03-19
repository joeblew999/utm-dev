# ADR-001: Align UTM Storage with SDK Caching System

## Status

**Implemented** - UTM storage aligned with utm-dev SDK caching for idempotency.

## Context

UTM previously used repo-local paths (`.bin/UTM.app`, `.data/utm/`) while Android SDKs use global paths (`~/utm-dev-sdks/`). This inconsistency meant:
- ISOs were re-downloaded per project
- No checksum verification for cached files
- Different caching patterns confused users

## Decision

All UTM paths are now global (shared across projects):

| Component | Legacy (Local) | Current (Global) |
|-----------|----------------|------------------|
| UTM.app | `.bin/UTM.app` | `~/utm-dev-sdks/utm/UTM.app` |
| ISOs | `.data/utm/iso/` | `~/utm-dev-sdks/utm/iso/` |
| VMs | `.data/utm/vms/` | `~/utm-dev-sdks/utm/vms/` |
| Share | `.data/utm/share/` | `~/utm-dev-sdks/utm/share/` |
| Cache | None | `~/utm-dev-cache/cache.json` |

**Rationale:** VMs are large (20-60GB) and reused across projects. Share folders are used for host<->VM file transfer and can be organized per-VM.

## Implementation Plan

### Phase 1: Update Path Configuration ✅

**File: `pkg/utm/config.go`**

`DefaultPaths()` uses global SDK paths for everything:

```go
func DefaultPaths() Paths {
    sdkDir := config.GetSDKDir()  // ~/utm-dev-sdks

    return Paths{
        // All paths are global (shared across projects)
        Root:  filepath.Join(sdkDir, "utm"),
        App:   filepath.Join(sdkDir, "utm", "UTM.app"),
        ISO:   filepath.Join(sdkDir, "utm", "iso"),
        VMs:   filepath.Join(sdkDir, "utm", "vms"),
        Share: filepath.Join(sdkDir, "utm", "share"),
    }
}
```

`GetUTMCtlPath()` checks global location first, then legacy local for migration.

### Phase 2: Cache Integration ✅

**File: `pkg/utm/cache.go`**

Integrated with existing `pkg/installer/cache.go`:

```go
// Cache key format
"utm-app-5.0.1"           // UTM application
"utm-iso-debian-13-arm"   // ISO files

// Functions
IsUTMAppCached(version, checksum) bool
IsISOCached(vmKey) bool
AddUTMAppToCache(version, checksum) error
AddISOToCache(vmKey) error
```

**File: `pkg/utm/install.go`**

`InstallUTM()` and `DownloadISO()` now:
1. Check cache first → return if cached
2. Download if needed
3. Add to cache after success

### Phase 3: Migration Support ✅

**File: `pkg/utm/migrate.go`**

```go
MigrateUTMApp()  // .bin/UTM.app → ~/utm-dev-sdks/utm/UTM.app
MigrateISOs()    // .data/utm/iso/* → ~/utm-dev-sdks/utm/iso/
MigrateAll()     // Full migration with status output
```

**File: `cmd/utm.go`**

Added `utm-dev utm migrate` command.

### Phase 4: Remove Taskfile.utm.yml ✅

Deleted `Taskfile.utm.yml` - all functionality in Go CLI:

| Taskfile Command | Go CLI Equivalent |
|------------------|-------------------|
| `task utm:install` | `utm-dev utm install` |
| `task utm:install:check` | `utm-dev utm doctor` |
| `task utm:vm:list` | `utm-dev utm list` |
| `task utm:vm:start` | `utm-dev utm start <vm>` |
| `task utm:gallery` | `utm-dev utm gallery` |
| (new) | `utm-dev utm migrate` |

### Phase 5: Global Paths for All ✅

All paths are now global:

```
~/utm-dev-sdks/utm/
├── UTM.app     # Application
├── iso/        # ISO images
├── vms/        # Virtual machines
└── share/      # Host<->VM file transfer
```

## Files to Modify

| File | Action | Description |
|------|--------|-------------|
| `pkg/utm/config.go` | Modify | Global paths for app/ISO |
| `pkg/utm/cache.go` | Create | Cache integration functions |
| `pkg/utm/install.go` | Modify | Add cache checks and updates |
| `pkg/utm/migrate.go` | Create | Migration logic |
| `cmd/utm.go` | Modify | Add `migrate` command |
| `pkg/utm/vm-gallery.json` | Modify | Update paths, add checksums |
| `Taskfile.utm.yml` | Delete | Replaced by Go CLI |

## Verification

1. **Test idempotency:**
   ```bash
   utm-dev utm install              # Downloads UTM
   utm-dev utm install              # Says "already cached"
   utm-dev utm install debian-13-arm # Downloads ISO
   utm-dev utm install debian-13-arm # Says "already cached"
   ```

2. **Verify cache:**
   ```bash
   cat ~/utm-dev-cache/cache.json | grep utm
   # Should show utm-app-* and utm-iso-* entries
   ```

3. **Test migration:**
   ```bash
   # If legacy files exist
   utm-dev utm migrate
   # Moves .bin/UTM.app and .data/utm/iso/* to global location
   ```

4. **Verify paths:**
   ```bash
   utm-dev utm paths
   # Shows: App: ~/utm-dev-sdks/utm/UTM.app
   #        ISO: ~/utm-dev-sdks/utm/iso
   ```

## Expected cache.json After Implementation

```json
{
  "entries": {
    "utm-app-5.0.1": {
      "name": "utm-app-5.0.1",
      "version": "5.0.1",
      "checksum": "sha256:...",
      "installPath": "/Users/apple/utm-dev-sdks/utm/UTM.app"
    },
    "utm-iso-debian-13-arm": {
      "name": "utm-iso-debian-13-arm",
      "version": "debian-13-arm",
      "checksum": "sha256:...",
      "installPath": "/Users/apple/utm-dev-sdks/utm/iso/debian-13-arm64.iso"
    }
  }
}
```

## Consequences

### Benefits
- Single caching system for all SDKs and tools
- ISOs and VMs shared across projects (no re-downloading, no duplicate VMs)
- Checksum verification prevents corruption
- Idempotent installs (fast, reliable)
- Simpler CLI-only workflow (no Taskfile needed)
- Large VMs (20-60GB) stored once, used everywhere

### Trade-offs
- Migration step required for existing users (one-time)
- Global paths mean disk space used even if project deleted
- Share folder organization is per-VM (e.g., `share/debian-13-arm/`)

## References

- `pkg/installer/cache.go` - Existing cache implementation
- `pkg/config/config.go` - GetSDKDir(), GetCacheDir()
- `Taskfile.utm.yml` - Removed (replaced by Go CLI)
