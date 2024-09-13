//go:build !unix

package slog_align

import "syscall"

// Don't do anything for now. Although I think Windows supports this in some
// way? Dunno.
var sigWinChange Signal = syscall.Signal(-1)
