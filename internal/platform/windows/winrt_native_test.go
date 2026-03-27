//go:build windows

package windows

import (
	"errors"
	"testing"

	"github.com/go-ole/go-ole"
)

func TestIsHResultNotFound(t *testing.T) {
	if !isHResultNotFound(ole.NewError(0x80070490)) {
		t.Fatalf("expected HRESULT 0x80070490 to be treated as not found")
	}
	if isHResultNotFound(ole.NewError(0x80004005)) {
		t.Fatalf("unexpected match for non not-found HRESULT")
	}
	if isHResultNotFound(errors.New("plain error")) {
		t.Fatalf("unexpected match for plain error")
	}
}

