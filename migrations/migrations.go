package migrations

import "embed"

// FS contains versioned, forward-only production schema migrations.
//
//go:embed *.sql
var FS embed.FS
