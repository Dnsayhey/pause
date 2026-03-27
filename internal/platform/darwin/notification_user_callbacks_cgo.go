//go:build darwin && cgo

package darwin

/*
 */
import "C"

import (
	"fmt"

	"pause/internal/logx"
)

//export pauseDarwinCaptureAuthorizationResult
func pauseDarwinCaptureAuthorizationResult(requestID C.int, granted C.int, errorMsg *C.char) {
	var err error
	if errorMsg != nil {
		err = fmt.Errorf("%s", C.GoString(errorMsg))
	}
	logx.Infof(
		"darwin.notification.authorization_request callback request_id=%d granted=%t has_error=%t",
		int(requestID),
		granted != 0,
		err != nil,
	)
	completeDarwinNotificationAuthorizationWaiter(int(requestID), darwinNotificationAuthorizationResult{
		granted: granted != 0,
		err:     err,
	})
}
