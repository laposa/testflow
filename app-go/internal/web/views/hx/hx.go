// Package hx writes htmx attributes for Gomponents (spec D3).
//
// It replaces gomponents-htmx, whose helpers target htmx 2. Attr works with any
// htmx attribute; the named helpers cover the attributes TestFlow uses. Only
// this file changes if TestFlow later moves to gomponents-htmx.
package hx

import g "maragu.dev/gomponents"

// Attr renders the htmx attribute hx-<name>="value".
func Attr(name, value string) g.Node { return g.Attr("hx-"+name, value) }

// Get issues a GET request to url.
func Get(url string) g.Node { return Attr("get", url) }

// Post issues a POST request to url.
func Post(url string) g.Node { return Attr("post", url) }

// Target selects the element that receives the response.
func Target(selector string) g.Node { return Attr("target", selector) }

// Swap sets how the response is swapped in, e.g. "outerHTML".
func Swap(strategy string) g.Node { return Attr("swap", strategy) }

// Trigger sets the event that issues the request, e.g. "every 30s".
func Trigger(spec string) g.Node { return Attr("trigger", spec) }

// Select picks part of the response to swap in.
func Select(selector string) g.Node { return Attr("select", selector) }

// PushURL pushes a URL into the browser history ("true", "false" or a URL).
func PushURL(v string) g.Node { return Attr("push-url", v) }

// Confirm asks the user to confirm before the request is sent.
func Confirm(message string) g.Node { return Attr("confirm", message) }

// Indicator selects the element that gets the htmx-request class while the
// request is in flight.
func Indicator(selector string) g.Node { return Attr("indicator", selector) }

// Sync coordinates requests between elements, e.g. "this:replace".
func Sync(spec string) g.Node { return Attr("sync", spec) }
