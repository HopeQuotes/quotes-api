package main

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/tomasen/realip"
	"golang.org/x/exp/slices"
	"golang.org/x/time/rate"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pascaldekloe/jwt"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				app.serverError(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		authorizationHeader := r.Header.Get("Authorization")

		if authorizationHeader != "" {
			headerParts := strings.Split(authorizationHeader, " ")

			if len(headerParts) == 2 && headerParts[0] == "Bearer" {
				token := headerParts[1]

				claims, err := jwt.HMACCheck([]byte(token), []byte(app.config.JWT.SecretKey))
				if err != nil {
					app.invalidAuthenticationToken(w, r)
					return
				}

				if !claims.Valid(time.Now()) {
					app.invalidAuthenticationToken(w, r)
					return
				}

				if claims.Issuer != app.config.BaseURL {
					app.invalidAuthenticationToken(w, r)
					return
				}

				if !claims.AcceptAudience(app.config.BaseURL) {
					app.invalidAuthenticationToken(w, r)
					return
				}

				userID, err := uuid.Parse(claims.Subject)
				if err != nil {
					app.serverError(w, r, err)
					return
				}

				user, err := app.db.GetUser(userID)
				if err != nil {
					app.serverError(w, r, err)
					return
				}

				if user != nil {
					r = contextSetAuthenticatedUser(r, user)
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) requireAuthenticatedUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authenticatedUser := contextGetAuthenticatedUser(r)

		if authenticatedUser == nil {
			app.authenticationRequired(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) rateLimit(next http.Handler) http.Handler {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app.config.Limiter.Enabled {
			ip := realip.FromRequest(r)

			mu.Lock()

			if _, found := clients[ip]; !found {
				clients[ip] = &client{
					limiter:  rate.NewLimiter(rate.Limit(app.config.Limiter.RPS), app.config.Limiter.Burst),
					lastSeen: time.Now(),
				}
			}

			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				app.rateLimitExceededResponse(w, r)
				return
			}

			mu.Unlock()
		}
		next.ServeHTTP(w, r)
	})
}

func (app *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")

		origin := r.Header.Get("Origin")

		if origin != "" && len(app.config.Cors.TrustedOrigins) != 0 {
			for i := range app.config.Cors.TrustedOrigins {
				if origin == app.config.Cors.TrustedOrigins[i] {
					w.Header().Set("Access-Control-Allow-Origin", origin)

					if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
						w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
						w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
						w.WriteHeader(http.StatusOK)
						return
					}
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) requirePermission(code string, next http.HandlerFunc) http.Handler {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := contextGetAuthenticatedUser(r)
		permissions, err := app.db.GetAllPermissionsForUser(user.ID)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		if !slices.Contains(permissions, code) {
			app.notPermittedResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
	return app.requireAuthenticatedUser(fn)
}
