package views

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// HomePage is a placeholder for GET / until the session list (iteration 6)
// takes over that route.
func HomePage(p Page) g.Node {
	return Layout(p,
		h.H1(g.Text("Test sessions")),
		Section("Coming soon", false,
			h.P(g.Text("The Go version of TestFlow is being built. Sessions will appear here.")),
			h.A(h.Href("/sessions/new"), h.Class("button filled"), g.Text("New session")),
		),
	)
}
