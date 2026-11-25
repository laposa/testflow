package views

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// LoginPage ports resources/views/auth/login.blade.php. message, if set, is
// shown above the button (e.g. after a failed sign-in).
func LoginPage(p Page, message string) g.Node {
	return Layout(p,
		h.Div(h.Class("login"),
			g.If(message != "", h.P(h.Class("login__error"), h.Role("alert"), g.Text(message))),
			h.P(g.Text("You need to login via your GitHub account.")),
			h.Form(h.Method("post"), h.Action("/auth/redirect"),
				h.Button(h.Type("submit"), h.Class("filled"), g.Text("Sign in with GitHub")),
			),
		),
	)
}
