package main

import (
	"errors"
	"github.com/alexedwards/flow"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
	"javlonrahimov/quotes-api/internal/database"
	"javlonrahimov/quotes-api/internal/filters"
	"javlonrahimov/quotes-api/internal/request"
	"javlonrahimov/quotes-api/internal/response"
	"javlonrahimov/quotes-api/internal/validator"
	"net/http"
	"time"
)

type QuoteResponse struct {
	ID        uuid.UUID         `json:"id,omitempty"`
	State     StateResponse     `json:"state,omitempty"`
	Author    string            `json:"author,omitempty"`
	Text      string            `json:"text,omitempty"`
	Hashtags  []HashtagResponse `json:"hashtags,omitempty"`
	CreatedBy uuid.UUID         `json:"createdBy,omitempty"`
	CreatedAt string            `json:"createdAt,omitempty"`
	UpdatedAt string            `json:"updatedAt,omitempty"`
	Photo     PhotoResponse     `json:"photo,omitempty"`
}

func (app *application) createQuote(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Author     *string             `json:"author,omitempty"`
		Text       *string             `json:"text,omitempty"`
		PhotoID    *uuid.UUID          `json:"photoID,omitempty"`
		HashtagIDs []uuid.UUID         `json:"hashtagIDs,omitempty"`
		StateID    *uuid.UUID          `json:"stateID,omitempty"`
		Validator  validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	input.Validator.CheckField(input.Author != nil && len(*input.Author) > 2, "author", "invalid author name")
	input.Validator.CheckField(input.Text != nil && len(*input.Text) > 10, "text", "min length must be at least 10 characters long")
	input.Validator.CheckField(input.HashtagIDs != nil && len(input.HashtagIDs) != 0, "hashtagIDs", "add at least one hashtag")

	input.Validator.CheckField(input.PhotoID != nil, "photoID", "required field")

	if input.Validator.HasErrors() {
		app.failedValidation(w, r, input.Validator)
		return
	}

	user := contextGetAuthenticatedUser(r)
	if user == nil {
		app.serverError(w, r, err)
		return
	}

	if slices.Contains(app.config.Sudoers, user.Email) {
		input.StateID = nil
	}

	quote, err := app.db.InsertQuote(*input.Author, *input.Text, user.ID, *input.PhotoID, input.HashtagIDs, input.StateID)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrRecordNotFound):
			app.notFound(w, r)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	var hashtags []HashtagResponse
	for _, hashtag := range quote.Hashtags {
		hashtags = append(hashtags, HashtagResponse{
			ID:    hashtag.ID,
			Value: hashtag.Value,
		})
	}

	data := QuoteResponse{
		ID: quote.ID,
		State: StateResponse{
			ID:        quote.State.ID,
			Value:     quote.State.Value,
			IdDefault: quote.State.IsDefault,
			Color:     quote.State.Color,
			IsPublic:  quote.State.IsPublic,
		},
		Author:    quote.Author,
		Text:      quote.Text,
		Hashtags:  hashtags,
		CreatedBy: quote.CreatedBy,
		CreatedAt: quote.CreatedAt.Format(time.RFC3339),
		UpdatedAt: quote.UpdatedAt.Format(time.RFC3339),
		Photo: PhotoResponse{
			ID:       quote.Photo.ID,
			Color:    quote.Photo.Color,
			BlurHash: quote.Photo.BlurHash,
			Author:   quote.Photo.Author,
			Url:      quote.Photo.Url,
		},
	}

	err = response.JSON(w, http.StatusOK, getWrapper(data))
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) updateQuote(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Author     *string             `json:"author"`
		Text       *string             `json:"text"`
		PhotoID    *uuid.UUID          `json:"photoID"`
		HashtagIDs []uuid.UUID         `json:"hashtagIDs"`
		Validator  validator.Validator `json:"-"`
	}

	quoteID, err := uuid.Parse(flow.Param(r.Context(), "id"))
	if err != nil {
		app.errorMessage(w, r, http.StatusNotFound, "quote not found", nil)
		return
	}

	err = request.DecodeJSON(w, r, &input)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	input.Validator.CheckField(input.Author != nil && len(*input.Author) > 2, "author", "invalid author name")
	input.Validator.CheckField(input.Author != nil && len(*input.Text) > 10, "text", "min length must be at least 10 characters long")
	input.Validator.CheckField(len(input.HashtagIDs) != 0, "hashtagIDs", "add at least one hashtag")

	input.Validator.CheckField(input.PhotoID.String() != "", "photoID", "required field")

	if input.Validator.HasErrors() {
		app.failedValidation(w, r, input.Validator)
		return
	}

	user := contextGetAuthenticatedUser(r)
	if user == nil {
		app.serverError(w, r, err)
		return
	}

	quote, err := app.db.GetQuoteById(quoteID)
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

	quote, err = app.db.UpdateQuote(quote.ID, *input.PhotoID, *input.Author, *input.Text, input.HashtagIDs)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrEditConflict):
			app.editConflictResponse(w, r)
			return
		default:
			app.serverError(w, r, err)
			return
		}
	}

	hashtags := []HashtagResponse{}
	for _, hashtag := range quote.Hashtags {
		hashtags = append(hashtags, HashtagResponse{
			ID:    hashtag.ID,
			Value: hashtag.Value,
		})
	}

	data := QuoteResponse{
		ID: quote.ID,
		State: StateResponse{
			ID:        quote.State.ID,
			Value:     quote.State.Value,
			IdDefault: quote.State.IsDefault,
			IsPublic:  quote.State.IsPublic,
		},
		Author:    quote.Author,
		Text:      quote.Text,
		Hashtags:  hashtags,
		CreatedBy: quote.CreatedBy,
		CreatedAt: quote.CreatedAt.Format(time.RFC3339),
		UpdatedAt: quote.UpdatedAt.Format(time.RFC3339),
		Photo: PhotoResponse{
			ID:       quote.Photo.ID,
			Color:    quote.Photo.Color,
			BlurHash: quote.Photo.BlurHash,
			Author:   quote.Photo.Author,
			Url:      quote.Photo.Url,
		},
	}

	err = response.JSON(w, http.StatusOK, getWrapper(data))
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) getQuoteById(w http.ResponseWriter, r *http.Request) {
	quoteID, err := uuid.Parse(flow.Param(r.Context(), "id"))
	if err != nil {
		app.errorMessage(w, r, http.StatusNotFound, "quote not found", nil)
		return
	}

	quote, err := app.db.GetQuoteById(quoteID)
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

	var hashtags []HashtagResponse
	for _, hashtag := range quote.Hashtags {
		hashtags = append(hashtags, HashtagResponse{
			ID:    hashtag.ID,
			Value: hashtag.Value,
		})
	}

	data := QuoteResponse{
		ID: quote.ID,
		State: StateResponse{
			ID:        quote.State.ID,
			Value:     quote.State.Value,
			IdDefault: quote.State.IsDefault,
			Color:     quote.State.Color,
			IsPublic:  quote.State.IsPublic,
		},
		Author:    quote.Author,
		Text:      quote.Text,
		Hashtags:  hashtags,
		CreatedBy: quote.CreatedBy,
		CreatedAt: quote.CreatedAt.Format(time.RFC3339),
		UpdatedAt: quote.UpdatedAt.Format(time.RFC3339),
	}

	err = response.JSON(w, http.StatusOK, getWrapper(data))
	if err != nil {
		app.serverError(w, r, err)
	}

}

func (app *application) deleteQuoteById(w http.ResponseWriter, r *http.Request) {
	quoteID, err := uuid.Parse(flow.Param(r.Context(), "id"))
	if err != nil {
		app.errorMessage(w, r, http.StatusNotFound, "quote not found", nil)
		return
	}

	err = app.db.DeleteQuoteById(quoteID)
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
}

func (app *application) getUserQuotes(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Text   string
		Author string
		State  uuid.UUID
		filters.Filters
		Validator validator.Validator
	}

	userId, err := uuid.Parse(flow.Param(r.Context(), "userId"))
	if err != nil {
		app.errorMessage(w, r, http.StatusNotFound, "user not found", nil)
		return
	}

	qs := r.URL.Query()
	input.Author = request.ReadString(qs, "author", "")
	input.Text = request.ReadString(qs, "text", "")

	input.Filters.Page = request.ReadInt(qs, "page", 1, &input.Validator)
	input.Filters.PageSize = request.ReadInt(qs, "page_size", 20, &input.Validator)

	input.Filters.Sort = request.ReadString(qs, "sort", "id")

	input.Filters.SortSafeList = []string{"id", "text", "date", "-id", "-text", "-date"}

	quotes, metadata, err := app.db.GetUserQuotes(userId, input.Author, input.Text, input.State, input.Filters)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	quotesResponse := make([]QuoteResponse, 0)
	for _, quote := range quotes {
		var hashtags []HashtagResponse
		for _, hashtag := range quote.Hashtags {
			hashtags = append(hashtags, HashtagResponse{
				ID:    hashtag.ID,
				Value: hashtag.Value,
			})
		}
		quotesResponse = append(quotesResponse, QuoteResponse{
			ID: quote.ID,
			State: StateResponse{
				ID:        quote.State.ID,
				Value:     quote.State.Value,
				IdDefault: quote.State.IsDefault,
				Color:     quote.State.Color,
				IsPublic:  quote.State.IsPublic,
			},
			Author:    quote.Author,
			Text:      quote.Text,
			Hashtags:  hashtags,
			CreatedBy: quote.CreatedBy,
			CreatedAt: quote.CreatedAt.Format(time.RFC3339),
			UpdatedAt: quote.UpdatedAt.Format(time.RFC3339),
			Photo: PhotoResponse{
				ID:       quote.Photo.ID,
				Color:    quote.Photo.Color,
				BlurHash: quote.Photo.BlurHash,
				Author:   quote.Photo.Author,
				Url:      quote.Photo.Url,
			},
		})
	}

	err = response.JSON(w, http.StatusOK, getWrapper(envelope{"metadata": metadata, "data": quotesResponse}))
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) getQuotes(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Text   string
		Author string
		filters.Filters
		Validator validator.Validator
	}

	qs := r.URL.Query()
	input.Author = request.ReadString(qs, "author", "")
	input.Text = request.ReadString(qs, "text", "")

	input.Filters.Page = request.ReadInt(qs, "page", 1, &input.Validator)
	input.Filters.PageSize = request.ReadInt(qs, "page_size", 20, &input.Validator)

	input.Filters.Sort = request.ReadString(qs, "sort", "created_at")

	input.Filters.SortSafeList = []string{"text", "created_at", "-text", "-created_at"}

	quotes, metadata, err := app.db.GetQuotes(input.Author, input.Text, input.Filters)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	quotesResponse := make([]QuoteResponse, 0)
	for _, quote := range quotes {
		var hashtags []HashtagResponse
		for _, hashtag := range quote.Hashtags {
			hashtags = append(hashtags, HashtagResponse{
				ID:    hashtag.ID,
				Value: hashtag.Value,
			})
		}
		quotesResponse = append(quotesResponse, QuoteResponse{
			ID: quote.ID,
			State: StateResponse{
				ID:        quote.State.ID,
				Value:     quote.State.Value,
				IdDefault: quote.State.IsDefault,
				Color:     quote.State.Color,
				IsPublic:  quote.State.IsPublic,
			},
			Author:    quote.Author,
			Text:      quote.Text,
			Hashtags:  hashtags,
			CreatedBy: quote.CreatedBy,
			CreatedAt: quote.CreatedAt.Format(time.RFC3339),
			UpdatedAt: quote.UpdatedAt.Format(time.RFC3339),
			Photo: PhotoResponse{
				ID:       quote.Photo.ID,
				Color:    quote.Photo.Color,
				BlurHash: quote.Photo.BlurHash,
				Author:   quote.Photo.Author,
				Url:      quote.Photo.Url,
			},
		})
	}

	err = response.JSON(w, http.StatusOK, getWrapper(envelope{"metadata": metadata, "data": quotesResponse}))
	if err != nil {
		app.serverError(w, r, err)
	}
}
