package main

import (
	"fmt"
	"net/http"

	"github.com/alexedwards/flow"
)

const (
	uuidRegex = "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
)

func (app *application) routes() http.Handler {
	mux := flow.New()

	mux.NotFound = http.HandlerFunc(app.routeNotFound)
	mux.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowed)

	mux.Use(app.recoverPanic)
	mux.Use(app.enableCORS)
	mux.Use(app.rateLimit)

	mux.HandleFunc("/v1/status", app.status, "GET")

	mux.HandleFunc("/v1/register", app.register, "POST")
	mux.HandleFunc("/v1/verify", app.verify, "POST")
	mux.HandleFunc("/v1/login", app.login, "POST")
	mux.HandleFunc("/v1/forgot-password", app.forgotPassword, "POST")
	mux.HandleFunc("/v1/reset-password", app.resetPassword, "POST")

	mux.Group(func(mux *flow.Mux) {
		mux.Use(app.authenticate)
		mux.Use(app.requireAuthenticatedUser)

		// quotes
		mux.HandleFunc("/v1/quote", app.createQuote, "POST")
		mux.HandleFunc(fmt.Sprintf("/v1/quote/:id|%s", uuidRegex), app.updateQuote, "PUT")
		mux.HandleFunc(fmt.Sprintf("/v1/quote/:id|%s", uuidRegex), app.getQuoteById, "GET")
		mux.HandleFunc(fmt.Sprintf("/v1/quote/:id|%s", uuidRegex), app.deleteQuoteById, "DELETE")
		mux.HandleFunc(fmt.Sprintf("/v1/quote/:userId|%s", uuidRegex), app.getUserQuotes, "GET")
		mux.HandleFunc("/v1/quotes", app.getQuotes, "GET")

		// quote states
		mux.HandleFunc("/v1/quote/states", app.getQuoteStates, "GET")
		mux.HandleFunc("/v1/quote/state", app.createQuoteState, "POST")
		mux.HandleFunc(fmt.Sprintf("/v1/quote/state/:id|%s", uuidRegex), app.deleteQuoteStateByID, "DELETE")
		mux.Handle("/v1/quote/state", app.requirePermission("quotes:state", app.setDefaultQuoteState), "PATCH")
		mux.Handle("/v1/quote/set-state", app.requirePermission("quotes:state", app.setQuoteState), "PATCH")

		//photos
		mux.HandleFunc("/v1/photo", app.createPhoto, "POST")
		mux.HandleFunc(fmt.Sprintf("/v1/photo/:id|%s", uuidRegex), app.getPhotoById, "GET")
		mux.HandleFunc("/v1/photos", app.getPhotos, "GET")

		// hashtags
		mux.HandleFunc("/v1/hashtag", app.createHashtag, "POST")
		mux.HandleFunc(fmt.Sprintf("/v1/hashtag/:id|%s", uuidRegex), app.deleteHashtagById, "DELETE")
		mux.HandleFunc(fmt.Sprintf("/v1/hashtags/:id|%s", uuidRegex), app.getQuoteHashtags, "GET")
		mux.HandleFunc("/v1/hashtags", app.getHashtags, "GET")
	})

	return mux
}
