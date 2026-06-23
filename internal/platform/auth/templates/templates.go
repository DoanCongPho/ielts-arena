// Package templates embeds the auth HTML entry pages (/login, /pending) and
// their compiled CSS. We keep templates here (not under a global views dir)
// because they are owned by platform/auth and serve only the pre-SPA flow.
package templates

import "embed"

//go:embed *.html login.css
var FS embed.FS
