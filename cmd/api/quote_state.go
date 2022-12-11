package main

import (
	"errors"
	"github.com/alexedwards/flow"
	"github.com/google/uuid"
	"javlonrahimov/quotes-api/internal/database"
	"javlonrahimov/quotes-api/internal/request"
	"javlonrahimov/quotes-api/internal/response"
	"javlonrahimov/quotes-api/internal/util"
	"javlonrahimov/quotes-api/internal/validator"
	"net/http"
)

type StateResponse struct {
	ID        uuid.UUID `json:"id,omitempty"`
	Value     string    `json:"value,omitempty"`
	IdDefault bool      `json:"idDefault"`
	Color     string    `json:"color"`
}

func (app *application) createQuoteState(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Value     string              `json:"value"`
		IsDefault bool                `json:"isDefault"`
		Color     string              `json:"color"`
		Validator validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	input.Validator.CheckField(input.Value != "", "value", "invalid value")
	input.Validator.CheckField(input.Color != "", "color", "required field")
	input.Validator.CheckField(validator.IsValidHexColor(input.Color), "color", "invalid color")

	if input.Validator.HasErrors() {
		app.failedValidation(w, r, input.Validator)
		return
	}

	state := &database.QuoteState{Value: input.Value, IsDefault: input.IsDefault, Color: input.Color}

	err = app.db.InsertQuoteState(state)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrDuplicateQuoteState):
			input.Validator.AddFieldError("value", "state already exists")
			app.failedValidation(w, r, input.Validator)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	err = response.JSON(w, http.StatusOK, getWrapper(envelope{"stateID": state.ID}))
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) setDefaultQuoteState(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ID        uuid.UUID           `json:"id"`
		Validator validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	input.Validator.CheckField(input.ID.String() != "", "id", "id is required")

	if input.Validator.HasErrors() {
		app.failedValidation(w, r, input.Validator)
		return
	}

	err = app.db.SetDefaultQuoteState(input.ID)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrRecordNotFound):
			app.notFound(w, r)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	err = response.JSON(w, http.StatusOK, getWrapper(envelope{"stateID": input.ID}))
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) deleteQuoteStateByID(w http.ResponseWriter, r *http.Request) {
	stateID, err := uuid.Parse(flow.Param(r.Context(), "id"))
	if err != nil {
		app.errorMessage(w, r, http.StatusNotFound, "user not found", nil)
		return
	}

	if exists := app.db.IsExistsWithThisState(stateID); exists {
		app.errorMessage(w, r, http.StatusConflict, "there one or more quotes with this state", nil)
		return
	}

	err = app.db.DeleteQuoteStateById(stateID)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrDefaultState):
			app.errorMessage(w, r, http.StatusNotFound, "cannot delete default state", nil)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	err = response.JSON(w, http.StatusOK, getWrapper(envelope{"stateID": stateID.String()}))
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) getQuoteStates(w http.ResponseWriter, r *http.Request) {
	states, metadata, err := app.db.GetAllQuoteStates()
	if err != nil {
		app.logger.Info(err.Error())
		app.serverError(w, r, err)
		return
	}

	data := util.Map(states, func(t *database.QuoteState) StateResponse {
		return StateResponse{
			ID:        t.ID,
			Value:     t.Value,
			IdDefault: t.IsDefault,
			Color:     t.Color,
		}
	})

	err = response.JSON(w, http.StatusOK, getWrapper(envelope{"metadata": metadata, "data": data}))
	if err != nil {
		app.logger.Info(err.Error())
		app.serverError(w, r, err)
	}
}

func (app *application) setQuoteState(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ID        uuid.UUID           `json:"id"`
		StateID   uuid.UUID           `json:"stateId"`
		Validator validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	input.Validator.CheckField(input.ID.String() != "", "id", "id is required")
	input.Validator.CheckField(input.StateID.String() != "", "stateId", "stateId is required")

	if input.Validator.HasErrors() {
		app.failedValidation(w, r, input.Validator)
		return
	}

	err = app.db.SetQuoteState(input.ID, input.StateID)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrRecordNotFound):
			app.notFound(w, r)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	err = response.JSON(w, http.StatusOK, getWrapper(envelope{"stateID": input.StateID}))
	if err != nil {
		app.serverError(w, r, err)
	}
}
