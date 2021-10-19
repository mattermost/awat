// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"

	"github.com/aws/aws-sdk-go/aws/session"
	cloudModel "github.com/mattermost/mattermost-cloud/model"
	"github.com/sirupsen/logrus"
)

// Context provides the API with all necessary data and interfaces for responding to requests.
//
// It is cloned before each request, allowing per-request changes such as logger annotations.
type Context struct {
	Store     Store
	Logger    logrus.FieldLogger
	AWS       *AWSContext
	Workdir   string
	RequestID string
}

type AWSContext struct {
	Session *session.Session
	Bucket  string
}

// Clone creates a shallow copy of context, allowing clones to apply per-request changes.
func (c *Context) Clone() *Context {
	return &Context{
		Store:   c.Store,
		Logger:  c.Logger,
		AWS:     c.AWS,
		Workdir: c.Workdir,
	}
}

type contextHandlerFunc func(c *Context, w http.ResponseWriter, r *http.Request)

type contextHandler struct {
	context *Context
	handler contextHandlerFunc
}

// ServeHTTP satisfies the Handler interface for contextHandler
func (h contextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	context := h.context.Clone()
	context.RequestID = cloudModel.NewID()
	context.Logger = context.Logger.WithFields(
		logrus.Fields{
			"path":    r.URL.Path,
			"request": context.RequestID,
		})

	h.handler(context, w, r)
}

func newContextHandler(context *Context, handler contextHandlerFunc) *contextHandler {
	return &contextHandler{
		context: context,
		handler: handler,
	}
}
