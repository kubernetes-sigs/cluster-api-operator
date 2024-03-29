//go:build tools
// +build tools

package tools

// This file tracks some external tools we use during development and release
// processes. These are not used at runtime but having them here allows the
// Go toolchain to see that we need to include them in go.mod and go.sum.

import (
	_ "golang.org/x/tools/cmd/stringer"
)
