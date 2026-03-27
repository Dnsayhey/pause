//go:build darwin && cgo

package darwin

/*
 */
import "C"

import (
	"fmt"
)

//export pauseDarwinCaptureAuthorizationResult
func pauseDarwinCaptureAuthorizationResult(requestID C.int, granted C.int, errorMsg *C.char) {
	var err error
	if errorMsg != nil {
		err = fmt.Errorf("%s", C.GoString(errorMsg))
	}
	completeDarwinNotificationAuthorizationWaiter(int(requestID), darwinNotificationAuthorizationResult{
		granted: granted != 0,
		err:     err,
	})
}
