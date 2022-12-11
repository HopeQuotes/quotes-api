package main

import (
	"javlonrahimov/quotes-api/internal/response"
	"net/http"
)

func (app *application) status(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"status": "OK",
	}

	err := response.JSON(w, http.StatusOK, getWrapper(data))
	if err != nil {
		app.serverError(w, r, err)
	}
}
