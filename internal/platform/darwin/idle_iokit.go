//go:build darwin

package darwin

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation

#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/IOKitLib.h>
#include <stdint.h>
#include <string.h>

#if __MAC_OS_X_VERSION_MAX_ALLOWED >= 120000
#define PAUSE_IO_PORT kIOMainPortDefault
#else
#define PAUSE_IO_PORT kIOMasterPortDefault
#endif

static int pauseReadHIDIdleTime(uint64_t *outNs) {
	if (outNs == NULL) {
		return -1;
	}

	io_registry_entry_t entry = IOServiceGetMatchingService(PAUSE_IO_PORT, IOServiceMatching("IOHIDSystem"));
	if (entry == IO_OBJECT_NULL) {
		return -2;
	}

	CFTypeRef value = IORegistryEntryCreateCFProperty(entry, CFSTR("HIDIdleTime"), kCFAllocatorDefault, 0);
	IOObjectRelease(entry);
	if (value == NULL) {
		return -3;
	}

	uint64_t ns = 0;
	int ok = 0;
	CFTypeID typeID = CFGetTypeID(value);
	if (typeID == CFNumberGetTypeID()) {
		if (CFNumberGetValue((CFNumberRef)value, kCFNumberSInt64Type, &ns)) {
			ok = 1;
		}
	} else if (typeID == CFDataGetTypeID()) {
		CFDataRef data = (CFDataRef)value;
		CFIndex length = CFDataGetLength(data);
		if (length >= (CFIndex)sizeof(uint64_t)) {
			memcpy(&ns, CFDataGetBytePtr(data), sizeof(uint64_t));
			ok = 1;
		}
	}

	CFRelease(value);
	if (!ok) {
		return -4;
	}
	*outNs = ns;
	return 0;
}
*/
import "C"

func queryDarwinIdleNanoseconds() (uint64, bool) {
	var idleNs C.uint64_t
	if C.pauseReadHIDIdleTime(&idleNs) != 0 {
		return 0, false
	}
	return uint64(idleNs), true
}
