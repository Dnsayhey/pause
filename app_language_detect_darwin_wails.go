//go:build darwin && wails

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation

#include <stdlib.h>

#import <Foundation/Foundation.h>

const char *PausePreferredLanguage(void) {
    @autoreleasepool {
        NSArray<NSString *> *languages = [NSLocale preferredLanguages];
        if (languages == nil || [languages count] == 0) {
            return NULL;
        }
        NSString *primary = [languages objectAtIndex:0];
        if (primary == nil || [primary length] == 0) {
            return NULL;
        }
        return strdup([primary UTF8String]);
    }
}
*/
import "C"

import "unsafe"

func detectPreferredLanguage() string {
	ptr := C.PausePreferredLanguage()
	if ptr == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(ptr))
	return C.GoString(ptr)
}
