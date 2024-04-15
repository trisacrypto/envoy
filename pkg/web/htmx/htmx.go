package htmx

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// HTMX Request Headers
const (
	HXBoosted               = "HX-Boosted"                 // indicates that the request is via an element using hx-boost
	HXCurrentURL            = "HX-Current-URL"             // the current URL of the browser
	HXHistoryRestoreRequest = "HX-History-Restore-Request" // “true” if the request is for history restoration after a miss in the local history cache
	HXPrompt                = "HX-Prompt"                  // the user response to an hx-prompt
	HXRequest               = "HX-Request"                 // always “true”
	HXtarget                = "HX-Target"                  // the id of the target element if it exists
	HXTriggerName           = "HX-Trigger-Name"            // the name of the triggered element if it exists
)

// HTMX Response Headers
const (
	HXLocation           = "HX-Location"             // allows you to do a client-side redirect that does not do a full page reload
	HxPushURL            = "HX-Push-Url"             // pushes a new url into the history stack
	HXRedirect           = "HX-Redirect"             // can be used to do a client-side redirect to a new location
	HXRefresh            = "HX-Refresh"              // if set to “true” the client-side will do a full refresh of the page
	HXReplaceURL         = "HX-Replace-Url"          // replaces the current URL in the location bar
	HXReswap             = "HX-Reswap"               // allows you to specify how the response will be swapped. See hx-swap for possible values
	HXRetarget           = "HX-Retarget"             //a CSS selector that updates the target of the content update to a different element on the page
	HXReselect           = "HX-Reselect"             // a CSS selector that allows you to choose which part of the response is used to be swapped in. Overrides an existing hx-select on the triggering element
	HXTriggerAfterSettle = "HX-Trigger-After-Settle" // allows you to trigger client-side events after the settle step
	HXTriggerAfterSwap   = "HX-Trigger-After-Swap"   // allows you to trigger client-side events after the swap step
)

// HXTrigger is both a request and a response header
const HXTrigger = "HX-Trigger" // the id of the triggered element if it exists in requests, allows you to trigger client-side events in responses

// Redirect determines if the request is an HTMX request, if so, it sets the HX-Redirect
// header and returns a 204 no content to allow HTMX to handle the redirect. Otherwise
// it sets the code and issues a normal gin redirect with the location in the headers.
func Redirect(c *gin.Context, code int, location string) {
	if IsHTMXRequest(c) {
		c.Header(HXRedirect, location)
		c.Status(http.StatusNoContent)
		return
	}

	c.Redirect(code, location)
}

func IsHTMXRequest(c *gin.Context) bool {
	return strings.ToLower(c.GetHeader(HXRequest)) == "true"
}
