package main

import (
	"fmt"
	"net/http"
	"strings"

	"javlonrahimov/quotes-api/internal/response"
	"javlonrahimov/quotes-api/internal/validator"
)

func (app *application) errorMessage(w http.ResponseWriter, r *http.Request, status int, message string, headers http.Header) {
	message = strings.ToUpper(message[:1]) + message[1:]

	err := response.JSONWithHeaders(w, status, getWrapper(envelope{"error": message}, "Failed"), headers)
	if err != nil {
		app.logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Error(err)

	message := "The server encountered a problem and could not process your request"
	app.errorMessage(w, r, http.StatusInternalServerError, message, nil)
}

func (app *application) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	message := "unable to update the record due to an edit conflict, please try again"
	app.errorMessage(w, r, http.StatusConflict, message, nil)
}

func (app *application) notFound(w http.ResponseWriter, r *http.Request) {
	message := "The requested resource could not be found"
	app.errorMessage(w, r, http.StatusNotFound, message, nil)
}

func (app *application) routeNotFound(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("Route not found: %s", r.URL)
	app.errorMessage(w, r, http.StatusNotFound, message, nil)
}

func (app *application) methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("The %s method is not supported for this resource", r.Method)
	app.errorMessage(w, r, http.StatusMethodNotAllowed, message, nil)
}

func (app *application) badRequest(w http.ResponseWriter, r *http.Request, err error) {
	app.errorMessage(w, r, http.StatusBadRequest, err.Error(), nil)
}

func (app *application) failedValidation(w http.ResponseWriter, r *http.Request, v validator.Validator) {
	err := response.JSON(w, http.StatusUnprocessableEntity, getWrapper(v, "Failed"))
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) invalidAuthenticationToken(w http.ResponseWriter, r *http.Request) {
	headers := make(http.Header)
	headers.Set("WWW-Authenticate", "Bearer")

	app.errorMessage(w, r, http.StatusUnauthorized, "Invalid authentication token", headers)
}

func (app *application) authenticationRequired(w http.ResponseWriter, r *http.Request) {
	app.errorMessage(w, r, http.StatusUnauthorized, "You must be authenticated to access this resource", nil)
}

func (app *application) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	message := "rate limit exceeded"
	app.errorMessage(w, r, http.StatusTooManyRequests, message, nil)
}

func (app *application) invalidOTP(w http.ResponseWriter, r *http.Request) {
	message := "invalid OTP"
	app.errorMessage(w, r, http.StatusBadRequest, message, nil)
}

func (app *application) notPermittedResponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account doesn't have the necessary permission to access this resource"
	app.errorMessage(w, r, http.StatusForbidden, message, nil)
}
