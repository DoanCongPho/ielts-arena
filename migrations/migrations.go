// Package migrations embeds the SQL migration files so the binary ships
// with them. The single FS variable is consumed by the platform/database
// migrate runner. Placed in this directory because Go's embed cannot use
// `..` parent paths.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
