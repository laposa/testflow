package views

import (
	"strconv"
	"time"

	"github.com/laposa/testflow/app-go/internal/store"
	"github.com/laposa/testflow/app-go/internal/web/views/hx"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ActivityView is the activity block of the session page: the comment form
// and the feed (spec §8.9).
type ActivityView struct {
	SessionID int64
	Entries   []store.Activity // newest first
	Now       time.Time        // for relative times
	Comment   string           // the comment kept after a validation error
	Errors    []string         // validation errors of the comment form
}

// ActivityID is the id of the activity block, which htmx replaces after a
// comment is posted.
const ActivityID = "activity"

// Activity ports livewire/activity/activity-list.blade.php and
// livewire/comments/create-comment.blade.php. The form is a normal POST
// form; htmx posts it and swaps in the returned block instead. It sits
// above the feed because the feed is newest first.
func Activity(a ActivityView) g.Node {
	action := "/sessions/" + strconv.FormatInt(a.SessionID, 10) + "/comments"
	return h.Section(h.ID(ActivityID), h.Class("activity"),
		h.Form(h.Method("post"), h.Action(action), h.Class("create-comment"),
			hx.Post(action), hx.Target("#"+ActivityID), hx.Swap("outerHTML"), hx.Sync("this:drop"),
			h.Div(h.Class("form-group"),
				h.Label(h.For("comment-body"), g.Text("Add a comment")),
				h.Textarea(h.ID("comment-body"), h.Name("body"), h.Rows("10"), h.Required(),
					g.If(len(a.Errors) > 0, g.Group{g.Attr("aria-invalid", "true"), g.Attr("aria-describedby", "comment-errors")}),
					g.Text(a.Comment)),
				g.If(len(a.Errors) > 0,
					h.Div(h.ID("comment-errors"), h.Class("field-error"), h.Role("alert"),
						g.Map(a.Errors, func(e string) g.Node { return h.P(g.Text(e)) }),
					),
				),
			),
			h.Button(h.Type("submit"), h.Class("button filled"), g.Text("Save")),
		),
		h.Div(h.Class("activity-feed"),
			g.Map(a.Entries, func(e store.Activity) g.Node { return activityItem(e, a.Now) }),
		),
	)
}

func activityItem(e store.Activity, now time.Time) g.Node {
	when := RelativeTime(e.CreatedAt, now)
	class := h.Class("activity-feed-item activity-feed--" + string(e.Type))
	if e.Type == store.ActivityComment {
		name := e.UserName
		if name == "" {
			name = "Unknown user"
		}
		return h.Div(class,
			h.Div(h.Class("activity-feed-header"),
				h.Div(h.Class("activity-feed-header-timestamp"), when),
				h.Strong(g.Text(name)), g.Text(" commented"),
			),
			// No Markdown (D9): the text keeps its line breaks through CSS.
			h.Div(h.Class("activity-feed-body activity-feed-text"), g.Text(e.Body)),
		)
	}
	return h.Div(class,
		h.Div(h.Class("activity-feed-header"), activityIcon(e.Type)),
		h.Div(h.Class("activity-feed-body"),
			h.Div(h.Class("activity-feed-body-timestamp"), when),
			g.Text(e.Body),
		),
	)
}

// RelativeTime renders t as "5 minutes ago" in a <time> element, with the
// exact time as its tooltip.
func RelativeTime(t, now time.Time) g.Node {
	return h.Time(h.DateTime(t.UTC().Format(time.RFC3339)), h.Title(t.Format("2 Jan 2006 15:04")),
		g.Text(Ago(now.Sub(t))))
}

// Ago describes a duration in the past, e.g. "3 hours ago". A month counts
// as 30 days and a year as 365.
func Ago(d time.Duration) string {
	const day = 24 * time.Hour
	units := []struct {
		size      time.Duration
		one, many string
	}{
		{365 * day, "year", "years"},
		{30 * day, "month", "months"},
		{7 * day, "week", "weeks"},
		{day, "day", "days"},
		{time.Hour, "hour", "hours"},
		{time.Minute, "minute", "minutes"},
	}
	for _, u := range units {
		if d >= u.size {
			return Plural(int(d/u.size), u.one, u.many) + " ago"
		}
	}
	return "just now"
}

// activityIcon returns the legacy icon of an activity type. The SVGs are
// constants, never user content; currentColor lets the CSS colour them.
func activityIcon(t store.ActivityType) g.Node {
	svg, ok := activityIcons[t]
	if !ok {
		return nil
	}
	return g.Raw(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" ` +
		`stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">` + svg + `</svg>`)
}

// activityIcons are the icon paths from activity-list.blade.php.
var activityIcons = map[store.ActivityType]string{
	store.ActivityReviewRequested: `<path d="M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.025 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 5.25h.008v.008H12v-.008Z"/>`,
	store.ActivityReviewApproved:  `<path d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"/>`,
	store.ActivityReviewRejected:  `<path d="m9.75 9.75 4.5 4.5m0-4.5-4.5 4.5M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"/>`,
	store.ActivitySessionCreated:  `<circle cx="12" cy="12" r="9"/><path d="M12 8v8m4-4H8"/>`,
	store.ActivityRunDispatched:   `<circle cx="12" cy="12" r="9"/><path d="M10 8l6 4-6 4V8z"/>`,
	store.ActivityRunStatusChanged: `<path stroke-width="2" d="M18.4721 16.7023C17.3398 18.2608 15.6831 19.3584 13.8064 19.7934C11.9297 20.2284 9.95909 19.9716 8.25656 19.0701C6.55404 18.1687 5.23397 16.6832 4.53889 14.8865C3.84381 13.0898 3.82039 11.1027 4.47295 9.29011C5.12551 7.47756 6.41021 5.96135 8.09103 5.02005C9.77184 4.07875 11.7359 3.77558 13.6223 4.16623C15.5087 4.55689 17.1908 5.61514 18.3596 7.14656C19.5283 8.67797 20.1052 10.5797 19.9842 12.5023M19.9842 12.5023L21.4842 11.0023M19.9842 12.5023L18.4842 11.0023"/>` +
		`<path stroke-width="2" d="M12 8V12L15 15"/>`,
	store.ActivityManualRunStarted: `<path stroke-width="2" d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle stroke-width="2" cx="9" cy="7" r="4"/>` +
		`<line stroke-width="2" x1="19" x2="19" y1="8" y2="14"/><line stroke-width="2" x1="22" x2="16" y1="11" y2="11"/>`,
	store.ActivityManualRunFinished: `<path stroke-width="2" d="M2 21a8 8 0 0 1 13.292-6"/><circle stroke-width="2" cx="10" cy="8" r="5"/><path stroke-width="2" d="m16 19 2 2 4-4"/>`,
}
