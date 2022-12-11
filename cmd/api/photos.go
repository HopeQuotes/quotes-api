package main

import (
	"errors"
	"github.com/alexedwards/flow"
	"github.com/google/uuid"
	"javlonrahimov/quotes-api/internal/database"
	"javlonrahimov/quotes-api/internal/request"
	"javlonrahimov/quotes-api/internal/response"
	"javlonrahimov/quotes-api/internal/validator"
	"net/http"
)

type PhotoResponse struct {
	ID       uuid.UUID `json:"id,omitempty"`
	Color    string    `json:"color,omitempty"`
	BlurHash string    `json:"blurHash,omitempty"`
	Author   string    `json:"author,omitempty"`
	Url      string    `json:"url,omitempty"`
}

func (app *application) createPhoto(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Color     *string             `json:"color,omitempty"`
		BlurHash  *string             `json:"blurHash,omitempty"`
		Author    *string             `json:"author,omitempty"`
		Url       *string             `json:"url,omitempty"`
		Validator validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	input.Validator.CheckField(input.Color != nil && len(*input.Color) == 7, "color", "invalid color")
	input.Validator.CheckField(input.BlurHash != nil && *input.BlurHash != "", "blurHash", "required field")
	input.Validator.CheckField(input.Author != nil && len(*input.Author) > 2, "author", "must be longer than 2")
	input.Validator.CheckField(input.Url != nil && *input.Url != "", "url", "required field")

	if input.Validator.HasErrors() {
		app.failedValidation(w, r, input.Validator)
		return
	}

	photo, err := app.db.InsertPhoto(*input.Color, *input.BlurHash, *input.Author, *input.Url)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := PhotoResponse{
		ID:       photo.ID,
		Color:    photo.Color,
		BlurHash: photo.BlurHash,
		Author:   photo.Author,
		Url:      photo.Url,
	}

	err = response.JSON(w, http.StatusOK, getWrapper(data))
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) getPhotoById(w http.ResponseWriter, r *http.Request) {
	photoID, err := uuid.Parse(flow.Param(r.Context(), "id"))
	if err != nil {
		app.errorMessage(w, r, http.StatusNotFound, "quote not found", nil)
		return
	}

	photo, err := app.db.GetPhotoById(photoID)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrRecordNotFound):
			app.notFound(w, r)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	data := PhotoResponse{
		ID:       photo.ID,
		Color:    photo.Color,
		BlurHash: photo.BlurHash,
		Author:   photo.Author,
		Url:      photo.Url,
	}

	err = response.JSON(w, http.StatusOK, getWrapper(data))
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) getPhotos(w http.ResponseWriter, r *http.Request) {
	photos, metadata, err := app.db.GetPhotos()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	var data []PhotoResponse
	for _, photo := range photos {
		data = append(data, PhotoResponse{
			ID:       photo.ID,
			Color:    photo.Color,
			BlurHash: photo.BlurHash,
			Author:   photo.Author,
			Url:      photo.Url,
		})
	}

	err = response.JSON(w, http.StatusOK, getWrapper(envelope{"metadata": metadata, "data": data}))
	if err != nil {
		app.serverError(w, r, err)
	}
}
