# Example Gio App

This is a simple example application built with the [Gio](https://gioui.org) UI library.

## Building and Running

To build and run this application, you will need to have the `utm-dev` tool installed and configured. You can then use the `Taskfile.yml` in the root of this repository to build and run the application for different platforms.

### macOS

To build and run the application for macOS, run the following commands:

```
task build:gio:macos
task run:gio:macos
```

### Android

To build the application for Android, run the following command:

```
task build:gio:android
```

This will generate an `.apk` file in this directory. You can then install this on an Android device or emulator.

### iOS

To build the application for iOS, run the following command:

```
task build:gio:ios
```

This will generate an `.app` bundle in this directory. You will need a provisioning profile to run this on an iOS device.

### Windows

To build the application for Windows, run the following command:

```
task build:gio:windows
```

This will generate an `.exe` file in the `.bin` directory. You can then run this on a Windows machine.
