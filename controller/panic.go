// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package controller

import (
	"runtime/debug"

	"github.com/xuhn/optimusprime/log"
)

// PanicFilter wraps the action invocation in a protective defer blanket that
// converts panics into 500 error pages.
func PanicFilter(c *Controller, fc []Filter) {
	defer func() {
		if err := recover(); err != nil {
			handleInvocationPanic(c, err)
		}
	}()
	fc[0](c, fc[1:])
}

// This function handles a panic in an action invocation.
// It cleans up the stack trace, logs it, and displays an error page.
func handleInvocationPanic(c *Controller, err interface{}) {
	error := NewErrorFromPanic(err)
	if error == nil {
		// Only show the sensitive information in the debug stack trace in development mode, not production
		log.ERROR(err, "\n", string(debug.Stack()))
		c.Response.Out.WriteHeader(500)
		_, _ = c.Response.Out.Write(debug.Stack())
		return
	}

	log.ERROR(err, "\n", error.Stack)
	c.Result = c.RenderError(error)
}
