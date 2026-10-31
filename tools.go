//go:build tools

// ponytail: tools.go pins 4 future-phase deps (gopher-lua, pty, x/crypto/ssh, x/sys)
// so go mod tidy keeps them. Delete this file when real code imports them.
package tools

import (
	_ "github.com/creack/pty"
	_ "github.com/yuin/gopher-lua"
	_ "golang.org/x/crypto/ssh"
	_ "golang.org/x/sys/unix"
)
