# CI/CD Integration

utm-dev is designed for seamless integration with continuous integration and deployment pipelines.

## GitHub Actions

### Basic Workflow

```yaml
name: Cross-Platform Build

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.19
    
    - name: Install Task
      run: sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d
    
    - name: Setup utm-dev
      run: |
        task build
        task setup:android
    
    - name: Build for all platforms
      run: |
        utm-dev build android ./my-app
        utm-dev build linux ./my-app
        utm-dev build web ./my-app
    
    - name: Upload artifacts
      uses: actions/upload-artifact@v3
      with:
        name: builds
        path: my-app/.bin/
```

### iOS Builds (macOS Runner)

```yaml
  build-ios:
    runs-on: macos-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.19
    
    - name: Setup iOS development
      run: |
        task build
        task setup:ios
    
    - name: Build iOS app
      run: utm-dev build ios ./my-app
```

### Windows Builds

```yaml
  build-windows:
    runs-on: windows-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.19
    
    - name: Setup Windows development
      run: |
        task build
        task setup:windows
    
    - name: Build Windows app
      run: utm-dev build windows-msix ./my-app
```

## GitLab CI

```yaml
stages:
  - build
  - test
  - deploy

variables:
  GO_VERSION: "1.19"

before_script:
  - apt-get update -qq && apt-get install -y -qq git curl
  - curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
  - sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d

build:android:
  stage: build
  image: golang:${GO_VERSION}
  script:
    - task build
    - task setup:android
    - utm-dev build android ./my-app
  artifacts:
    paths:
      - my-app/.bin/
    expire_in: 1 week

build:web:
  stage: build
  image: golang:${GO_VERSION}
  script:
    - task build
    - utm-dev build web ./my-app
  artifacts:
    paths:
      - my-app/.bin/
    expire_in: 1 week
```

## Docker Support

### Multi-stage Build

```dockerfile
# Build stage
FROM golang:1.19-alpine AS builder

RUN apk add --no-cache git curl bash

# Install Task
RUN sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d

WORKDIR /app
COPY . .

# Build utm-dev and setup environment
RUN task build
RUN task setup:android

# Build the application
RUN utm-dev build android ./my-app
RUN utm-dev build linux ./my-app

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy built applications
COPY --from=builder /app/my-app/.bin/ ./builds/

CMD ["./builds/my-app-linux"]
```

### Development Container

```dockerfile
FROM golang:1.19

# Install dependencies
RUN apt-get update && apt-get install -y \
    curl \
    git \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install Task
RUN sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d

# Setup working directory
WORKDIR /workspace

# Pre-setup common SDKs
COPY . .
RUN task build
RUN task setup:android

CMD ["bash"]
```

## Environment Variables

Configure builds using environment variables:

```bash
# SDK paths
export ANDROID_SDK_ROOT=/opt/android-sdk
export ANDROID_NDK_ROOT=/opt/android-ndk

# Build configuration
export GOUP_CACHE_DIR=/tmp/goup-cache
export GOUP_SDK_DIR=/opt/sdks

# Platform-specific settings
export IOS_TEAM_ID=YOUR_TEAM_ID
export WINDOWS_CERT_THUMBPRINT=YOUR_CERT
```

## Build Optimization

### Caching

```yaml
# GitHub Actions caching
- name: Cache SDKs
  uses: actions/cache@v3
  with:
    path: |
      sdks/
      ~/.cache/utm-dev
    key: ${{ runner.os }}-sdks-${{ hashFiles('**/sdk-*.json') }}
    restore-keys: |
      ${{ runner.os }}-sdks-
```

### Parallel Builds

```bash
# Build multiple platforms in parallel
utm-dev build android ./my-app &
utm-dev build linux ./my-app &
utm-dev build web ./my-app &
wait
```

## Deployment Integration

### Automated Releases

```yaml
- name: Create Release
  if: startsWith(github.ref, 'refs/tags/')
  uses: softprops/action-gh-release@v1
  with:
    files: |
      my-app/.bin/*.apk
      my-app/.bin/*.exe
      my-app/.bin/*.app
      my-app/.bin/*.msix
```

### Store Deployment

```bash
# Upload to Google Play
utm-dev deploy android ./my-app --store google-play

# Upload to App Store
utm-dev deploy ios ./my-app --store app-store

# Upload to Microsoft Store
utm-dev deploy windows ./my-app --store microsoft-store
```

## Monitoring & Notifications

### Build Status

```bash
# Slack notification on build completion
curl -X POST -H 'Content-type: application/json' \
    --data '{"text":"Build completed for all platforms!"}' \
    $SLACK_WEBHOOK_URL
```

### Error Reporting

```bash
# Capture build logs
utm-dev build all ./my-app 2>&1 | tee build.log

# Send error reports
if [ $? -ne 0 ]; then
    # Send to error tracking service
    curl -X POST $ERROR_ENDPOINT -d @build.log
fi
```
