//go:build unix

package slog_align

import "syscall"

var sigWinChange = syscall.SIGWINCH
