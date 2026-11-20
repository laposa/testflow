package views

import (
	"github.com/laposa/testflow/app-go/internal/store"
	"github.com/laposa/testflow/app-go/internal/web/views/hx"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ReviewsID is the reviews block of the session page, which htmx replaces
// after a review action. ReviewErrorsID is its message area.
const (
	ReviewsID      = "reviews"
	ReviewErrorsID = "review-errors"
)

// ReviewsView is the reviews block: the pending requests and the form to
// request one (spec §8.9).
type ReviewsView struct {
	SessionID  int64
	UserID     int64 // the signed-in user
	Pending    []store.ReviewRequest
	Candidates []store.User // users without a pending request
	Errors     []string
}

// Reviews ports the review parts of sessions/show.blade.php and
// livewire/reviews/*: the reviewer of a pending request gets Approve and
// Reject, its requester Revoke, everyone else a note. Every form is a
// normal POST form; htmx replaces this block with the answer.
func Reviews(v ReviewsView) g.Node {
	base := "/sessions/" + id(v.SessionID) + "/reviews"
	return h.Section(h.ID(ReviewsID), h.Class("reviews"),
		h.Div(h.ID(ReviewErrorsID), h.Class("field-error"), h.Role("alert"), g.Map(v.Errors, ErrorText)),
		g.Map(v.Pending, func(r store.ReviewRequest) g.Node {
			url := base + "/" + id(r.ID)
			if r.Reviewer.ID == v.UserID {
				return h.Div(h.Class("review-form"),
					h.Div(h.Class("review-form-icon"), g.Raw(reviewIcon)),
					h.Div(h.Class("review-form-title"),
						h.H3(g.Text("Review Required")),
						h.P(g.Text(r.Requester.Name+" requested a review")),
					),
					h.Div(h.Class("review-form-actions"),
						reviewButton(url+"/approve", "button filled", "Approve"),
						reviewButton(url+"/reject", "button danger", "Reject"),
					),
				)
			}
			return h.Div(h.Class("review-form"),
				h.P(g.Text("Review pending for "+r.Reviewer.Name)),
				g.If(r.Requester.ID == v.UserID, h.Div(h.Class("review-form-actions"),
					reviewButton(url+"/revoke", "button", "Revoke"),
				)),
			)
		}),
		h.Form(h.Method("post"), h.Action(base), h.Class("request-review"),
			hx.Post(base), hx.Target("#"+ReviewsID), hx.Swap("outerHTML"), hx.Sync("this:drop"),
			h.Select(h.Name("reviewer_id"), h.Required(), g.Attr("aria-label", "Reviewer"),
				h.Option(h.Value(""), g.Text("Select a reviewer")),
				g.Map(v.Candidates, func(u store.User) g.Node { return h.Option(h.Value(id(u.ID)), g.Text(u.Name)) }),
			),
			h.Button(h.Type("submit"), h.Class("button filled"), g.Text("Request")),
		),
	)
}

// ReviewsWithFeed is the answer to an htmx review action: the reviews
// block, and the activity feed out of band for the new entry.
func ReviewsWithFeed(v ReviewsView, a ActivityView) g.Node {
	return g.Group{Reviews(v), ActivityFeed(a, hx.Attr("swap-oob", "true"))}
}

func reviewButton(action, class, label string) g.Node {
	return h.Form(h.Method("post"), h.Action(action),
		hx.Post(action), hx.Target("#"+ReviewsID), hx.Swap("outerHTML"), hx.Sync("this:drop"),
		h.Button(h.Type("submit"), h.Class(class), g.Text(label)),
	)
}

// reviewIcon is the legacy x-circle of review-form.blade.php.
const reviewIcon = `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" aria-hidden="true">` +
	`<path stroke-linecap="round" stroke-linejoin="round" d="m9.75 9.75 4.5 4.5m0-4.5-4.5 4.5M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"/></svg>`
