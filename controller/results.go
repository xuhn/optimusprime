// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package controller

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	_ "errors"
	"fmt"
	"io"
	_ "io/ioutil"
	"net/http"
	_ "reflect"
	"strconv"
	_ "strings"
	"time"

	"optimusprime/common"
	"optimusprime/log"
	"optimusprime/net/websocket"
)

type Result interface {
	Apply(req *Request, resp *Response)
}

// ErrorResult structure used to handles all kinds of error codes (500, 404, ..).
// It renders the relevant error page (errors/CODE.format, e.g. errors/500.json).
// If RunMode is "dev", this results in a friendly error page.
type ErrorResult struct {
	//	ViewArgs map[string]interface{}
	Error error
}

func (r ErrorResult) Apply(req *Request, resp *Response) {
	format := req.Format
	status := resp.Status
	if status == 0 {
		status = http.StatusInternalServerError
	}

	contentType := ContentTypeByFilename("xxx." + format)
	if contentType == DefaultFileContentType {
		contentType = "text/plain"
	}
	// If it's not a revel error, wrap it in one.
	var revelError *Error
	switch e := r.Error.(type) {
	case *Error:
		revelError = e
	case error:
		revelError = &Error{
			Title:       "Server Error",
			Description: e.Error(),
		}
	}

	if revelError == nil {
		panic("no error provided")
	}
	var b bytes.Buffer
	// need to check if we are on a websocket here
	// net/http panics if we write to a hijacked connection
	if req.Method == "WS" {
		if err := websocket.Message.Send(req.Websocket, fmt.Sprint(revelError)); err != nil {
			log.ERRORF("Send failed:", err)
		}
	} else {
		resp.WriteHeader(status, contentType)
		if _, err := b.WriteTo(resp.Out); err != nil {
			log.ERRORF("Response WriteTo failed:", err)
		}
	}

}

type PlaintextErrorResult struct {
	Error error
}

// Apply method is used when the template loader or error template is not available.
func (r PlaintextErrorResult) Apply(req *Request, resp *Response) {
	resp.WriteHeader(http.StatusInternalServerError, "text/plain; charset=utf-8")
	if _, err := resp.Out.Write([]byte(r.Error.Error())); err != nil {
		log.ERRORF("Write error:", err)
	}
}

type RenderHTMLResult struct {
	html string
}

func (r RenderHTMLResult) Apply(req *Request, resp *Response) {
	resp.WriteHeader(http.StatusOK, "text/html; charset=utf-8")
	if _, err := resp.Out.Write([]byte(r.html)); err != nil {
		log.ERRORF("Response write failed:", err)
	}
}

type RenderJSONResult struct {
	obj      interface{}
	callback string
}

func (r RenderJSONResult) Apply(req *Request, resp *Response) {
	var b []byte
	var err error
	if common.BoolDefault("results.pretty", false) {
		b, err = json.MarshalIndent(r.obj, "", "  ")
	} else {
		b, err = json.Marshal(r.obj)
	}

	if err != nil {
		ErrorResult{Error: err}.Apply(req, resp)
		return
	}

	if r.callback == "" {
		resp.WriteHeader(http.StatusOK, "application/json; charset=utf-8")
		if _, err = resp.Out.Write(b); err != nil {
			log.ERRORF("Response write failed:", err)
		}
		return
	}

	resp.WriteHeader(http.StatusOK, "application/javascript; charset=utf-8")
	if _, err = resp.Out.Write([]byte(r.callback + "(")); err != nil {
		log.ERRORF("Response write failed:", err)
	}
	if _, err = resp.Out.Write(b); err != nil {
		log.ERRORF("Response write failed:", err)
	}
	if _, err = resp.Out.Write([]byte(");")); err != nil {
		log.ERRORF("Response write failed:", err)
	}
}

type RenderXMLResult struct {
	obj interface{}
}

func (r RenderXMLResult) Apply(req *Request, resp *Response) {
	var b []byte
	var err error
	if common.BoolDefault("results.pretty", false) {
		b, err = xml.MarshalIndent(r.obj, "", "  ")
	} else {
		b, err = xml.Marshal(r.obj)
	}

	if err != nil {
		ErrorResult{Error: err}.Apply(req, resp)
		return
	}

	resp.WriteHeader(http.StatusOK, "application/xml; charset=utf-8")
	if _, err = resp.Out.Write(b); err != nil {
		log.ERRORF("Response write failed:", err)
	}
}

type RenderTextResult struct {
	text string
}

func (r RenderTextResult) Apply(req *Request, resp *Response) {
	resp.WriteHeader(http.StatusOK, "text/plain; charset=utf-8")
	if _, err := resp.Out.Write([]byte(r.text)); err != nil {
		log.ERRORF("Response write failed:", err)
	}
}

type ContentDisposition string

var (
	Attachment ContentDisposition = "attachment"
	Inline     ContentDisposition = "inline"
)

type BinaryResult struct {
	Reader   io.Reader
	Name     string
	Length   int64
	Delivery ContentDisposition
	ModTime  time.Time
}

func (r *BinaryResult) Apply(req *Request, resp *Response) {
	disposition := string(r.Delivery)
	if r.Name != "" {
		disposition += fmt.Sprintf(`; filename="%s"`, r.Name)
	}
	resp.Out.Header().Set("Content-Disposition", disposition)

	// If we have a ReadSeeker, delegate to http.ServeContent
	if rs, ok := r.Reader.(io.ReadSeeker); ok {
		// http.ServeContent doesn't know about response.ContentType, so we set the respective header.
		if resp.ContentType != "" {
			resp.Out.Header().Set("Content-Type", resp.ContentType)
		} else {
			contentType := ContentTypeByFilename(r.Name)
			resp.Out.Header().Set("Content-Type", contentType)
		}
		http.ServeContent(resp.Out, req.Request, r.Name, r.ModTime, rs)
	} else {
		// Else, do a simple io.Copy.
		if r.Length != -1 {
			resp.Out.Header().Set("Content-Length", strconv.FormatInt(r.Length, 10))
		}
		resp.WriteHeader(http.StatusOK, ContentTypeByFilename(r.Name))
		if _, err := io.Copy(resp.Out, r.Reader); err != nil {
			log.ERRORF("Response write failed:", err)
		}
	}

	// Close the Reader if we can
	if v, ok := r.Reader.(io.Closer); ok {
		_ = v.Close()
	}
}

type RedirectToURLResult struct {
	url string
}

func (r *RedirectToURLResult) Apply(req *Request, resp *Response) {
	resp.Out.Header().Set("Location", r.url)
	resp.WriteHeader(http.StatusFound, "")
}
