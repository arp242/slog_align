//go:build !unix

package slog_align

import (
	"os"
	"syscall"
)

// Don't do anything for now. Although I think Windows supports this in some
// way? Dunno.
var sigWinChange os.Signal = syscall.Signal(-1)
