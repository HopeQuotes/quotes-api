package main

import (
	"errors"
	"github.com/alexedwards/flow"
	"github.com/google/uuid"
	"javlonrahimov/quotes-api/internal/database"
	"javlonrahimov/quotes-api/internal/filters"
	"javlonrahimov/quotes-api/internal/request"
	"javlonrahimov/quotes-api/internal/response"
	"javlonrahimov/quotes-api/internal/validator"
	"net/http"
	"strings"
)

type HashtagResponse struct {
	ID    uuid.UUID `json:"id,omitempty"`
	Value string    `json:"value,omitempty"`
}

func (app *application) createHashtag(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Value     string              `json:"value"`
		Validator validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	input.Validator.CheckField(len(input.Value) > 2, "value", "min length must be at leas 3 characters long")

	if input.Validator.HasErrors() {
		app.failedValidation(w, r, input.Validator)
		return
	}

	hashtag, err := app.db.InsertHashtag(strings.ToLower(input.Value))
	if err != nil {
		switch {
		case errors.Is(err, database.ErrDuplicateHashtag):
			input.Validator.AddFieldError("value", "value already exists")
			app.failedValidation(w, r, input.Validator)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	err = response.JSON(w, http.StatusOK, getWrapper(HashtagResponse{
		ID:    hashtag.ID,
		Value: hashtag.Value,
	}))
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) deleteHashtagById(w http.ResponseWriter, r *http.Request) {
	hashtagID, err := uuid.Parse(flow.Param(r.Context(), "id"))
	if err != nil {
		app.errorMessage(w, r, http.StatusNotFound, "hashtag not found", nil)
		return
	}

	if exists := app.db.IsQuoteExistsWithThisHashtag(hashtagID); exists {
		app.errorMessage(w, r, http.StatusBadRequest, "there are quotes with this hashtag", nil)
		return
	}

	err = app.db.DeleteHashtagById(hashtagID)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrRecordNotFound):
			app.notFound(w, r)
			return
		default:
			app.serverError(w, r, err)
			return
		}
	}

	err = response.JSON(w, http.StatusOK, getWrapper(map[string]interface{}{
		"id": hashtagID,
	}))
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) getQuoteHashtags(w http.ResponseWriter, r *http.Request) {
	quoteID, err := uuid.Parse(flow.Param(r.Context(), "id"))
	if err != nil {
		app.errorMessage(w, r, http.StatusNotFound, "quote not found", nil)
		return
	}

	data, metadata, err := app.db.GetQuoteHashtags(quoteID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	hashtags := []HashtagResponse{}
	for _, hashtag := range data {
		hashtags = append(hashtags, HashtagResponse{
			ID:    hashtag.ID,
			Value: hashtag.Value,
		})
	}

	err = response.JSON(w, http.StatusOK, getWrapper(envelope{"metadata": metadata, "data": hashtags}))
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) getHashtags(w http.ResponseWriter, r *http.Request) {
	var input struct {
		filters.Filters
		Validator validator.Validator
	}
	qs := r.URL.Query()

	input.Filters.Page = request.ReadInt(qs, "page", 1, &input.Validator)
	input.Filters.PageSize = request.ReadInt(qs, "pageSize", 20, &input.Validator)

	input.Filters.Sort = request.ReadString(qs, "sort", "value")

	input.Filters.SortSafeList = []string{"value", "-value"}

	data, metadata, err := app.db.GetHashtags(input.Filters)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	hashtags := []HashtagResponse{}
	for _, hashtag := range data {
		hashtags = append(hashtags, HashtagResponse{
			ID:    hashtag.ID,
			Value: hashtag.Value,
		})
	}

	err = response.JSON(w, http.StatusOK, getWrapper(envelope{"metadata": metadata, "data": hashtags}))
	if err != nil {
		app.serverError(w, r, err)
	}
}
