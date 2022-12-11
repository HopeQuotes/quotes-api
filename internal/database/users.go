package database

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"time"
)

type User struct {
	ID             uuid.UUID `db:"id"`
	CreatedAt      time.Time `db:"created_at"`
	Email          string    `db:"email"`
	HashedPassword string    `db:"hashed_password"`
	Name           string    `db:"name"`
	IsActivated    bool      `db:"is_activated"`
}

func (db *DB) InsertUser(email, hashedPassword, name string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	user := &User{}

	query := `
		INSERT INTO users (id, created_at, email, hashed_password, name)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, email, name, is_activated`

	err := db.QueryRowContext(ctx, query, uuid.New(), time.Now(), email, hashedPassword, name).Scan(
		&user.ID, &user.CreatedAt, &user.Email, &user.Name, &user.IsActivated,
	)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return nil, ErrDuplicateEmail
		default:
			return nil, err
		}
	}

	return user, err
}

func (db *DB) GetUser(id uuid.UUID) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var user User

	query := `SELECT * FROM users WHERE id = $1`

	err := db.GetContext(ctx, &user, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return &user, err
}

func (db *DB) GetUserByEmail(email string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var user User

	query := `SELECT * FROM users WHERE email = $1`

	err := db.GetContext(ctx, &user, query, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRecordNotFound
	}

	return &user, err
}

func (db *DB) UpdateUserHashedPassword(id uuid.UUID, hashedPassword string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `UPDATE users SET hashed_password = $1 WHERE id = $2`

	_, err := db.ExecContext(ctx, query, hashedPassword, id)
	return err
}

func (db *DB) ActivateUser(id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `UPDATE users SET is_activated = true WHERE id = $1`

	_, err := db.ExecContext(ctx, query, id)
	return err
}
