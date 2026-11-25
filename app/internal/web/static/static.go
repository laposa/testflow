// Package static embeds the CSS, fonts, icons and JavaScript served under
// /static/ (spec §10). Third-party files are vendored and pinned (spec §2).
package static

import "embed"

// FS holds every static asset, at the same paths as in this directory.
//
//go:embed *.css *.js fonts icons
var FS embed.FS
