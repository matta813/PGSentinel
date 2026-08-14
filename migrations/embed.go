package migrations

import "embed"

// FS contains ordered SQLite schema migrations.
//
//go:embed *.sql
var FS embed.FS
