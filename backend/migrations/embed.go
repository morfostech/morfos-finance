// Package migrations embeds the versioned SQL files so the runner can apply
// them from the compiled binary without shipping the .sql files separately.
package migrations

import "embed"

//go:embed *.up.sql
var FS embed.FS
