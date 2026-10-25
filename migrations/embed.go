// Package migrations embeds the golang-migrate SQL files so the server binary
// can apply them on boot without shipping the directory separately.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
