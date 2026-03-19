# UTM AppleScript Automation

This document describes the UTM AppleScript automation patterns used in utm-dev, adapted from [naveenrajm7/packer-plugin-utm](https://github.com/naveenrajm7/packer-plugin-utm).

## Source Reference

The AppleScript automation code is cloned to `.src/packer-plugin-utm/` for reference:

```bash
git clone --depth 1 https://github.com/naveenrajm7/packer-plugin-utm .src/packer-plugin-utm
```

## Key Components

### AppleScript Files

Located in `pkg/utm/scripts/`:

| Script | Purpose |
|--------|---------|
| `create_vm.applescript` | Create empty VM with backend/arch |
| `customize_vm.applescript` | Set CPU, RAM, UEFI, hypervisor |
| `add_drive.applescript` | Add disk drive (interface + size) |
| `attach_iso.applescript` | Attach ISO (boot disk) |
| `add_network_interface.applescript` | Add network (shared/bridged) |
| `remove_drive.applescript` | Remove drive by ID |
| `clear_network_interfaces.applescript` | Clear all network interfaces |
| `add_port_forwards.applescript` | Configure port forwarding |

### Go Integration

The `pkg/utm/osascript.go` file provides:

1. **Embedded Scripts** via `//go:embed scripts/*`
2. **ExecuteOsaScript()** - Pipes embedded scripts to `osascript -`
3. **Enum Maps** - UTM AppleScript constant codes

### UTM Enum Codes

UTM uses 4-character enum codes in AppleScript:

**Backend:**
- `QeMu` - QEMU emulation
- `ApLe` - Apple Virtualization framework

**Controller/Interface:**
- `QdIv` - VirtIO
- `QdIu` - USB
- `QdIs` - SCSI
- `QdIi` - IDE

**Network Mode:**
- `ShRd` - Shared Network (NAT)
- `EmUd` - Emulated VLAN
- `BrDg` - Bridged
- `HsOn` - Host Only

## VM Creation Workflow

```bash
# 1. Create VM -> returns UUID
osascript create_vm.applescript --name "MyVM" --backend "QeMu" --arch "aarch64"

# 2. Configure hardware
osascript customize_vm.applescript <UUID> --cpus 2 --memory 2048 --uefi-boot true

# 3. Add disk (size in MB)
osascript add_drive.applescript <UUID> --interface "QdIv" --size 32768

# 4. Attach boot ISO
osascript attach_iso.applescript <UUID> --interface "QdIu" --source "/path/to/iso"

# 5. Add network
osascript add_network_interface.applescript <UUID> "ShRd"
```

## Usage in utm-dev

```bash
# Automated VM creation (recommended)
utm-dev utm create debian-13-arm

# With verbose output
utm-dev utm create debian-13-arm --verbose

# Force recreate existing VM
utm-dev utm create debian-13-arm --force

# Manual mode (shows instructions)
utm-dev utm create debian-13-arm --manual
```

## Prerequisites

1. **UTM Installed**: `utm-dev utm install`
2. **ISO Downloaded**: `utm-dev utm install <vm-key>`
3. **Automation Permission**: System Settings > Privacy > Automation > Terminal > UTM

## Troubleshooting

### "Can't make X into type constant"

The enum code is wrong. Check the UTM AppleScript dictionary for correct values.

### Permission Denied

Grant Automation permission in System Settings > Privacy & Security > Automation.

### UTM Not Found

Ensure UTM is installed and the path in `pkg/utm/config.go` is correct.

## References

- [UTM AppleScript Documentation](https://docs.getutm.app/scripting/cheat-sheet/)
- [Packer Plugin UTM](https://github.com/naveenrajm7/packer-plugin-utm)
- [UTM GitHub Issue #6691](https://github.com/utmapp/UTM/issues/6691) - Request for utmctl create
