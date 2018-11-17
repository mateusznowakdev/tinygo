// +build avr

package runtime

import (
	"unsafe"
)

const GOARCH = "avr"

// The bitness of the CPU (e.g. 8, 32, 64).
const TargetBits = 8

//go:extern _heap_start
var heapStartSymbol unsafe.Pointer

var heapStart = uintptr(unsafe.Pointer(&heapStartSymbol))

// Align on a word boundary.
func align(ptr uintptr) uintptr {
	// No alignment necessary on the AVR.
	return ptr
}
