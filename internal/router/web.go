package router

import (
	"context"
	"net/http"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/web"
	"github.com/swaggest/swgui/v3emb"
	"github.com/unrolled/secure"

	"github.com/go-chi/chi"
)

// WebHandler is an abstraction of an http handler that handles web calls.
type WebHandler interface {
	http.Handler
}

// NewWebHandler returns a new WebHandler.
func NewWebHandler(systemStore store.SystemStore) WebHandler {
	config := systemStore.Config(context.Background())

	// Use go-chi router for inner routing
	r := chi.NewRouter()
	// create middleware to enforce security best practices for
	// the user interface. note that theis middleware is only used
	// when serving the user interface (not found handler, below).
	sec := secure.New(
		secure.Options{
			AllowedHosts:          config.Secure.AllowedHosts,
			HostsProxyHeaders:     config.Secure.HostsProxyHeaders,
			SSLRedirect:           config.Secure.SSLRedirect,
			SSLTemporaryRedirect:  config.Secure.SSLTemporaryRedirect,
			SSLHost:               config.Secure.SSLHost,
			SSLProxyHeaders:       config.Secure.SSLProxyHeaders,
			STSSeconds:            config.Secure.STSSeconds,
			STSIncludeSubdomains:  config.Secure.STSIncludeSubdomains,
			STSPreload:            config.Secure.STSPreload,
			ForceSTSHeader:        config.Secure.ForceSTSHeader,
			FrameDeny:             config.Secure.FrameDeny,
			ContentTypeNosniff:    config.Secure.ContentTypeNosniff,
			BrowserXssFilter:      config.Secure.BrowserXSSFilter,
			ContentSecurityPolicy: config.Secure.ContentSecurityPolicy,
			ReferrerPolicy:        config.Secure.ReferrerPolicy,
		},
	)

	// openapi playground endpoints
	swagger := v3emb.NewHandler("API Definition", "/api/v1/swagger.yaml", "/swagger")
	r.With(sec.Handler).Handle("/swagger", swagger)
	r.With(sec.Handler).Handle("/swagger/*", swagger)

	// serve all other routes from the embedded filesystem,
	// which in turn serves the user interface.
	r.With(sec.Handler).NotFound(
		web.Handler(),
	)

	return r
}