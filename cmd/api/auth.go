package main

import (
	"errors"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
	"javlonrahimov/quotes-api/internal/database"
	"javlonrahimov/quotes-api/internal/security"
	"net/http"
	"time"

	o "javlonrahimov/quotes-api/internal/otp"
	"javlonrahimov/quotes-api/internal/password"
	"javlonrahimov/quotes-api/internal/request"
	"javlonrahimov/quotes-api/internal/response"
	"javlonrahimov/quotes-api/internal/validator"
)

type AuthResponse struct {
	UserID          uuid.UUID `json:"userID"`
	Name            string    `json:"name"`
	Email           string    `json:"email"`
	Created         string    `json:"created"`
	IsActivated     bool      `json:"isActivated"`
	AuthToken       string    `json:"authToken,omitempty"`
	AuthTokenExpiry string    `json:"authTokenExpiry,omitempty"`
}

func (app *application) register(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email     string              `json:"email"`
		Password  string              `json:"password"`
		Name      string              `json:"name"`
		Validator validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	input.Validator.CheckField(len(input.Name) > 2, "name", "name is too short")

	input.Validator.CheckField(input.Email != "", "email", "email is required")
	input.Validator.CheckField(validator.Matches(input.Email, validator.RgxEmail), "email", "Must be a valid email address")

	input.Validator.CheckField(input.Password != "", "password", "password is required")
	input.Validator.CheckField(len(input.Password) >= 8, "password", "password is too short")
	input.Validator.CheckField(len(input.Password) <= 72, "password", "password is too long")
	input.Validator.CheckField(validator.NotIn(input.Password, password.CommonPasswords...), "password", "password is too common")

	if input.Validator.HasErrors() {
		app.failedValidation(w, r, input.Validator)
		return
	}

	hashedPassword, err := password.Hash(input.Password)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	user, err := app.db.InsertUser(input.Email, hashedPassword, input.Name)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrDuplicateEmail):
			input.Validator.AddFieldError("email", "email is already in use")
			app.failedValidation(w, r, input.Validator)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	err = app.db.DeleteAllOTPForUser(user.ID, o.ScopeAuthentication)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	otp, err := app.db.NewOtp(user.ID, 20*time.Minute, o.ScopeAuthentication)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	go func() {
		err = app.mailer.Send(input.Email, otp.Plaintext, map[string]interface{}{
			"UserName":        input.Name,
			"ApplicationName": "Quotes",
			"OTP":             otp.Plaintext,
		}, "otp.tmpl")
		if err != nil {
			app.logger.Error(err)
			return
		}
	}()

	err = response.JSON(w, http.StatusOK, getWrapper(envelope{"userID": user.ID}))
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) verify(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email     string              `json:"email"`
		OTP       string              `json:"otp"`
		Validator validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	input.Validator.CheckField(input.Email != "", "email", "email is required")
	input.Validator.CheckField(validator.Matches(input.Email, validator.RgxEmail), "email", "Must be a valid email address")

	if input.Validator.HasErrors() {
		app.failedValidation(w, r, input.Validator)
		return
	}

	user, err := app.db.GetUserByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrRecordNotFound):
			app.notFound(w, r)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	otp, err := app.db.GetOTPForEmail(input.Email, o.ScopeAuthentication)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrRecordNotFound):
			app.invalidOTP(w, r)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	input.Validator.CheckField(input.Email != "", "email", "email is required")
	input.Validator.CheckField(user != nil, "email", "email address could not be found")

	if user != nil {

		otpMatches := o.Matches(input.OTP, otp.Hash)

		input.Validator.CheckField(input.OTP != "", "otp", "password is required")
		input.Validator.CheckField(otpMatches, "otp", "password is incorrect")
	}

	if input.Validator.HasErrors() {
		app.failedValidation(w, r, input.Validator)
		return
	}

	err = app.db.ActivateUser(user.ID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	user.IsActivated = true

	if slices.Contains(app.config.Sudoers, user.Email) {
		err = app.db.AddPermissionForUser(user.ID, "quotes:state")
		if err != nil {
			app.serverError(w, r, err)
			return
		}
	}

	expiry := time.Now().Add(7 * 24 * time.Hour)
	jwtString, err := security.NewJWT(user.ID, expiry, app.config.BaseURL, app.config.JWT.SecretKey)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := AuthResponse{
		UserID:          user.ID,
		Name:            user.Name,
		Email:           user.Email,
		Created:         user.CreatedAt.Format(time.RFC3339),
		IsActivated:     user.IsActivated,
		AuthToken:       jwtString,
		AuthTokenExpiry: expiry.Format(time.RFC3339),
	}

	err = app.db.DeleteAllOTPForUser(user.ID, o.ScopeAuthentication)
	if err != nil {
		app.logger.Error(err)
	}

	err = response.JSON(w, http.StatusOK, getWrapper(data))
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email     string              `json:"email"`
		Password  string              `json:"password"`
		Validator validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	user, err := app.db.GetUserByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrRecordNotFound):
			app.notFound(w, r)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	input.Validator.CheckField(input.Email != "", "email", "email is required")
	input.Validator.CheckField(user != nil, "email", "email address could not be found")

	if !user.IsActivated {
		err := app.db.DeleteAllOTPForUser(user.ID, o.ScopeAuthentication)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		otp, err := app.db.NewOtp(user.ID, 20*time.Minute, o.ScopeAuthentication)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		go func() {
			err = app.mailer.Send(input.Email, otp.Plaintext, map[string]interface{}{
				"UserName":        user.Name,
				"ApplicationName": "Quotes",
				"OTP":             otp.Plaintext,
			}, "otp.tmpl")

			if err != nil {
				app.logger.Error(err)
				return
			}
		}()

		data := AuthResponse{
			UserID:      user.ID,
			Name:        user.Name,
			Email:       user.Email,
			Created:     user.CreatedAt.Format(time.RFC3339),
			IsActivated: user.IsActivated,
		}

		err = response.JSON(w, http.StatusOK, getWrapper(data))
		if err != nil {
			app.serverError(w, r, err)
		}
		return
	}

	if user != nil {

		passwordMatches, err := password.Matches(input.Password, user.HashedPassword)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		input.Validator.CheckField(input.Password != "", "password", "password is required")
		input.Validator.CheckField(passwordMatches, "password", "password is incorrect")
	}

	if input.Validator.HasErrors() {
		app.failedValidation(w, r, input.Validator)
		return
	}
	expiry := time.Now().Add(7 * 24 * time.Hour)
	jwtString, err := security.NewJWT(user.ID, expiry, app.config.BaseURL, app.config.JWT.SecretKey)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := AuthResponse{
		UserID:          user.ID,
		Name:            user.Name,
		Email:           user.Email,
		Created:         user.CreatedAt.Format(time.RFC3339),
		IsActivated:     user.IsActivated,
		AuthToken:       jwtString,
		AuthTokenExpiry: expiry.Format(time.RFC3339),
	}

	err = response.JSON(w, http.StatusOK, getWrapper(data))
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email     string              `json:"email"`
		Validator validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	user, err := app.db.GetUserByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrRecordNotFound):
			app.notFound(w, r)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	input.Validator.CheckField(input.Email != "", "email", "email is required")
	input.Validator.CheckField(user != nil, "email", "email address could not be found")

	if input.Validator.HasErrors() {
		app.failedValidation(w, r, input.Validator)
		return
	}

	err = app.db.DeleteAllOTPForUser(user.ID, o.ScopeResetPassword)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	otp, err := app.db.NewOtp(user.ID, 20*time.Minute, o.ScopeResetPassword)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	go func() {
		err = app.mailer.Send(input.Email, otp.Plaintext, map[string]interface{}{
			"UserName":        user.Name,
			"ApplicationName": "Quotes",
			"OTP":             otp.Plaintext,
		}, "otp.tmpl")

		if err != nil {
			app.logger.Error(err)
			return
		}
	}()

	err = response.JSON(w, http.StatusOK, getWrapper(envelope{"email": input.Email}))
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) resetPassword(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email     string              `json:"email"`
		OTP       string              `json:"otp"`
		Password  string              `json:"password"`
		Validator validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(w, r, &input)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	input.Validator.CheckField(input.Email != "", "email", "email is required")
	input.Validator.CheckField(validator.Matches(input.Email, validator.RgxEmail), "email", "Must be a valid email address")

	input.Validator.CheckField(input.OTP != "", "otp", "otp is required")
	input.Validator.CheckField(len(input.OTP) == 4, "otp", "otp must be 4 characters long")

	input.Validator.CheckField(input.Password != "", "password", "password is required")
	input.Validator.CheckField(len(input.Password) >= 8, "password", "password is too short")
	input.Validator.CheckField(len(input.Password) <= 72, "password", "password is too long")
	input.Validator.CheckField(validator.NotIn(input.Password, password.CommonPasswords...), "password", "password is too common")

	if input.Validator.HasErrors() {
		app.failedValidation(w, r, input.Validator)
		return
	}

	user, err := app.db.GetUserByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrRecordNotFound):
			app.notFound(w, r)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	input.Validator.CheckField(user != nil, "email", "email address could not be found")

	if input.Validator.HasErrors() {
		app.failedValidation(w, r, input.Validator)
		return
	}

	otp, err := app.db.GetOTPForEmail(input.Email, o.ScopeResetPassword)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	input.Validator.CheckField(otp != nil, "otp", "otp must be provided")
	input.Validator.CheckField(len(otp.Plaintext) == 4, "otp", "otp must be 4 characters long")

	otpMatches := o.Matches(input.OTP, otp.Hash)

	input.Validator.CheckField(otpMatches, "otp", "otp is incorrect")

	if input.Validator.HasErrors() {
		app.failedValidation(w, r, input.Validator)
		return
	}

	hashedPassword, err := password.Hash(input.Password)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	err = app.db.UpdateUserHashedPassword(user.ID, hashedPassword)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	err = response.JSON(w, http.StatusOK, getWrapper(envelope{"userID": user.ID}))
	if err != nil {
		app.serverError(w, r, err)
	}
}
