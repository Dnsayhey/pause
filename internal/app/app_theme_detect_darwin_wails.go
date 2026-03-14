//go:build darwin && wails

package app

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Foundation

#include <stdlib.h>

#import <Cocoa/Cocoa.h>

const char *PausePreferredTheme(void) {
    @autoreleasepool {
        if (@available(macOS 10.14, *)) {
            NSAppearance *appearance = [NSApp effectiveAppearance];
            if (appearance != nil) {
                NSArray<NSAppearanceName> *names = @[NSAppearanceNameAqua, NSAppearanceNameDarkAqua];
                NSAppearanceName best = [appearance bestMatchFromAppearancesWithNames:names];
                if ([best isEqualToString:NSAppearanceNameDarkAqua]) {
                    return strdup("dark");
                }
                if ([best isEqualToString:NSAppearanceNameAqua]) {
                    return strdup("light");
                }
            }
        }
        return NULL;
    }
}
*/
import "C"

import "unsafe"

func detectPreferredTheme() string {
	ptr := C.PausePreferredTheme()
	if ptr == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(ptr))
	return C.GoString(ptr)
}
