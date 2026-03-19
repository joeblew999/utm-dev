// +build darwin

package screenshot

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation
#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>

// Get window bounds for a process by PID using CoreGraphics
// Returns 1 if found, 0 if not found
// Fills x, y, width, height with window bounds
// Finds the LARGEST window (by area) for the PID to skip menu bar items
int GetCGWindowBounds(int targetPID, int *x, int *y, int *width, int *height) {
    // Use kCGWindowListOptionAll to include Gio windows which aren't
    // visible with kCGWindowListOptionOnScreenOnly
    CFArrayRef windowList = CGWindowListCopyWindowInfo(
        kCGWindowListOptionAll,
        kCGNullWindowID
    );

    if (windowList == NULL) {
        return 0;
    }

    int found = 0;
    int bestArea = 0;
    CFIndex count = CFArrayGetCount(windowList);

    for (CFIndex i = 0; i < count; i++) {
        CFDictionaryRef window = (CFDictionaryRef)CFArrayGetValueAtIndex(windowList, i);

        // Get owner PID
        CFNumberRef pidRef = (CFNumberRef)CFDictionaryGetValue(window, kCGWindowOwnerPID);
        if (pidRef == NULL) continue;

        int pid = 0;
        CFNumberGetValue(pidRef, kCFNumberIntType, &pid);

        if (pid != targetPID) continue;

        // Found a window for this PID - get bounds
        CFDictionaryRef boundsRef = (CFDictionaryRef)CFDictionaryGetValue(window, kCGWindowBounds);
        if (boundsRef == NULL) continue;

        CGRect bounds;
        if (!CGRectMakeWithDictionaryRepresentation(boundsRef, &bounds)) continue;

        int w = (int)bounds.size.width;
        int h = (int)bounds.size.height;
        int area = w * h;

        // Skip very thin windows (menu bars are typically 33px tall)
        if (h < 50) continue;

        // Keep the largest window
        if (area > bestArea) {
            bestArea = area;
            *x = (int)bounds.origin.x;
            *y = (int)bounds.origin.y;
            *width = w;
            *height = h;
            found = 1;
        }
    }

    CFRelease(windowList);
    return found;
}

// Get window ID for a process by PID
// Returns window ID if found, 0 if not found
// Finds the LARGEST window (by area) for the PID to skip menu bar items
int GetCGWindowID(int targetPID) {
    // Use kCGWindowListOptionAll to include Gio windows
    CFArrayRef windowList = CGWindowListCopyWindowInfo(
        kCGWindowListOptionAll,
        kCGNullWindowID
    );

    if (windowList == NULL) {
        return 0;
    }

    int bestWindowID = 0;
    int bestArea = 0;
    CFIndex count = CFArrayGetCount(windowList);

    for (CFIndex i = 0; i < count; i++) {
        CFDictionaryRef window = (CFDictionaryRef)CFArrayGetValueAtIndex(windowList, i);

        // Get owner PID
        CFNumberRef pidRef = (CFNumberRef)CFDictionaryGetValue(window, kCGWindowOwnerPID);
        if (pidRef == NULL) continue;

        int pid = 0;
        CFNumberGetValue(pidRef, kCFNumberIntType, &pid);

        if (pid != targetPID) continue;

        // Get bounds to find the largest window
        CFDictionaryRef boundsRef = (CFDictionaryRef)CFDictionaryGetValue(window, kCGWindowBounds);
        if (boundsRef == NULL) continue;

        CGRect bounds;
        if (!CGRectMakeWithDictionaryRepresentation(boundsRef, &bounds)) continue;

        int h = (int)bounds.size.height;
        int w = (int)bounds.size.width;
        int area = w * h;

        // Skip very thin windows (menu bars are typically 33px tall)
        if (h < 50) continue;

        // Keep the window ID of the largest window
        if (area > bestArea) {
            bestArea = area;

            CFNumberRef windowNumberRef = (CFNumberRef)CFDictionaryGetValue(window, kCGWindowNumber);
            if (windowNumberRef == NULL) continue;

            CFNumberGetValue(windowNumberRef, kCFNumberIntType, &bestWindowID);
        }
    }

    CFRelease(windowList);
    return bestWindowID;
}
*/
import "C"

import (
	"fmt"
	"os/exec"
	"time"
)

// GetCGWindowBoundsByPID uses CoreGraphics to get window bounds for a PID
// This works for Gio windows which aren't visible to AppleScript
func GetCGWindowBoundsByPID(pid int) (int, int, int, int, error) {
	var x, y, width, height C.int

	found := C.GetCGWindowBounds(C.int(pid), &x, &y, &width, &height)
	if found == 0 {
		return 0, 0, 0, 0, fmt.Errorf("no window found for PID %d", pid)
	}

	return int(x), int(y), int(width), int(height), nil
}

// GetCGWindowIDByPID uses CoreGraphics to get window ID for a PID
func GetCGWindowIDByPID(pid int) (int, error) {
	windowID := C.GetCGWindowID(C.int(pid))
	if windowID == 0 {
		return 0, fmt.Errorf("no window found for PID %d", pid)
	}
	return int(windowID), nil
}

// WaitForCGWindow waits for a window to appear using CoreGraphics API
func WaitForCGWindow(pid int, timeout time.Duration) error {
	start := time.Now()

	for time.Since(start) < timeout {
		_, _, w, h, err := GetCGWindowBoundsByPID(pid)
		if err == nil && w > 0 && h > 0 {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("window for PID %d did not appear within %v", pid, timeout)
}

// CaptureWindowByCGBounds captures a window using macOS native screencapture.
// Uses screencapture -R (region) instead of -l (window ID) because -l cannot
// capture WKWebView content which renders in a separate compositor layer.
func CaptureWindowByCGBounds(pid int, output string, quality int) error {
	x, y, w, h, err := GetCGWindowBoundsByPID(pid)
	if err != nil {
		return err
	}

	// Use macOS native screencapture with region bounds
	// -R x,y,w,h = capture screen region (captures ALL layers including webviews)
	// -x = no shutter sound
	region := fmt.Sprintf("%d,%d,%d,%d", x, y, w, h)
	cmd := exec.Command("screencapture", "-R", region, "-x", output)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("screencapture failed: %w: %s", err, string(out))
	}
	return nil
}
