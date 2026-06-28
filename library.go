// This file marks the repo as a library, not a CLI application.
//
// Nothing ever sets the library_marker build tag, so this file never compiles —
// it just needs to exist. gomatic tooling and conventions tell a library repo
// apart from a CLI repo by whether this marker file is here, so don't remove it.

//go:build library_marker

package sql
