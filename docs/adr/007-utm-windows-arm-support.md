# ADR-007: UTM Windows ARM64 Support

## Status

**Implemented**

## Context

Windows 11 ARM64 is the primary development target for testing Gio/Tauri apps on Windows
from Apple Silicon. The initial approach (ISO + AppleScript VM creation) was wrong for
Windows. The correct approach uses a pre-built `.utm` box from naveenrajm7's HCP Vagrant
registry.

## Research Sources

- UTM source: `github.com/utmapp/UTM` (`Scripting/UTM.sdef`, `Configuration/`)
- naveenrajm7's ecosystem:
  - `github.com/naveenrajm7/packer-plugin-utm` — Packer plugin that builds the boxes
  - `github.com/naveenrajm7/utm-box` (bento fork) — Packer templates + `autounattend.xml`
  - `naveenrajm7.github.io/utm-gallery` — gallery + `getbox.sh` download script
  - HCP Vagrant registry: `app.vagrantup.com/utm/boxes/windows-11`

## Key Findings

### TPM cannot be set via AppleScript

`UTM.sdef` (the authoritative scripting dictionary) has NO `tpm` property on
`qemu configuration`. The internal Swift property `hasTPMDevice` in
`UTMQemuConfigurationQEMU.swift` is not exposed to AppleScript or `utmctl`.
The `--tpm` flag we added does nothing and was reverted.

### TPM is not needed — bypass via LabConfig registry keys

naveenrajm7's `autounattend.xml` (arm64) injects these registry keys during Windows PE
before the installer checks requirements:

```
HKLM\SYSTEM\Setup\LabConfig\BypassTPMCheck      = 1
HKLM\SYSTEM\Setup\LabConfig\BypassSecureBootCheck = 1
HKLM\SYSTEM\Setup\LabConfig\BypassStorageCheck   = 1
HKLM\SYSTEM\Setup\LabConfig\BypassCPUCheck       = 1
HKLM\SYSTEM\Setup\LabConfig\BypassRAMCheck       = 1
HKLM\SYSTEM\Setup\LabConfig\BypassDiskCheck      = 1
```

This means no TPM hardware or software TPM (swtpm) is required at all.

### Use the pre-built box, not ISO

naveenrajm7 already did all the hard work:
- Built Windows 11 24H2 ARM64 via Packer with full unattended install
- Loaded VirtIO drivers (storage, network, balloon, input, RNG, SCSI, serial, pvpanic)
- Installed UTM guest tools (`utm-guest-tools-0.229.exe`)
- Configured WinRM on port 5985
- Published as a `.box` on HCP Vagrant registry: `utm/windows-11` v0.0.0 arm64

The box is ~5.7 GB. It is downloaded, extracted (tar), and the `.utm` bundle inside is
imported via AppleScript — no OS installation needed.

### ISO SHA256 (for reference, if ever needed)

`57d1dfb2c6690a99fe99226540333c6c97d3fd2b557a50dfe3d68c3f675ef2b0`
(Win11_24H2_English_Arm64.iso)

### NVMe disk confirmed correct

naveenrajm7's packer template sets `hard_drive_interface = "nvme"` — confirms our fix
to use NVMe instead of virtio for Windows is correct.

### `utmctl --version` output format

Outputs exactly `5.0.2` (bare version, no prefix). The install version comparison in
`InstallUTM` is correct after trimming whitespace.

## Implementation

### `pkg/utm/box.go` (new)

Mirrors `getbox.sh`:
1. Hit HCP Vagrant API to resolve latest active version
2. Fetch signed download URL
3. Download `.box` with resumable Range headers (reuses `downloadFile`)
4. Extract tar to temp dir, find `.utm` directory
5. Import via `osascript -e 'tell application "UTM" to import new virtual machine from POSIX file "..."'`
6. Cache result, clean up `.box` file

### `pkg/utm/gallery.go`

- Added `BoxConfig` struct: `name` (namespace/box), `arch`, `checksum`, `size`
- Added `Box *BoxConfig` field to `VMEntry`
- Added `VMEntry.IsBoxBased()` helper

### `pkg/utm/vm-gallery.json`

- `windows-11-arm`: replaced `iso` with `box: {name: "utm/windows-11", arch: "arm64"}`

### `cmd/utm.go` — install command

Routes based on `vm.IsBoxBased()`:
- Box VM → `utm.InstallBox(vmKey, force)`
- ISO VM → `utm.DownloadISO(vmKey, force)`

### Reverted

- `--tpm` flag from `customize_vm.applescript` (not in sdef, does nothing)
- TPM field and logic from `create.go`
- `"tpm"` fields from `vm-gallery.json`

## End-to-end flow for Windows 11 ARM

```bash
utm-dev utm install              # install UTM v5.0.2
utm-dev utm install windows-11-arm  # download + import pre-built box (~5.7 GB)
utm-dev utm start "Windows 11"      # boot — already installed, no setup wizard
# RDP: localhost:3389  WinRM: localhost:5985  credentials: vagrant/vagrant
```

## Remaining gaps

- The `windows-11-x64` entry still uses ISO (no pre-built box exists for x64 emulated)
- Box checksum is empty (HCP registry returns `checksum_type: "NONE"` for this box)
- Port forwarding (RDP 3389, WinRM 5985) must be configured after import — the imported
  VM uses shared network; port forwards require switching to emulated VLAN network
