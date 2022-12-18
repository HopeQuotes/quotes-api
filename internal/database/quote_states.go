package database

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	f "javlonrahimov/quotes-api/internal/filters"
)

type QuoteState struct {
	ID        uuid.UUID `db:"id"`
	Value     string    `db:"value"`
	IsDefault bool      `db:"is_default"`
	Color     string    `db:"color"`
	IsPublic  bool      `db:"is_public"`
}

func (db *DB) InsertQuoteState(state *QuoteState) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
		insert into quote_states (id, value, is_default, color, is_public)
		values ($1, $2, $3, $4, $5)
		returning id`

	args := []interface{}{uuid.New(), state.Value, false, state.Color, state.IsPublic}

	err := db.QueryRowContext(ctx, query, args...).Scan(&state.ID)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "quote_states_value_key"`:
			return ErrDuplicateQuoteState
		default:
			return err
		}
	}

	if state.IsDefault {
		err := db.SetDefaultQuoteState(state.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) getDefaultQuoteState() (*QuoteState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var state QuoteState

	query := `
 		select id, value, is_default, color, is_public
		from quote_states
		where is_default = true`

	err := db.QueryRowContext(ctx, query).Scan(&state.ID, &state.Value, &state.IsDefault, &state.Color, &state.IsPublic)
	if err != nil {
		return nil, err
	}

	return &state, nil
}

func (db *DB) DeleteQuoteStateById(id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	if state, _ := db.getDefaultQuoteState(); state != nil && state.ID == id {
		return ErrDefaultState
	}

	query := `delete from quote_states where id = $1`

	_, err := db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) ExistsQuoteStateById(id uuid.UUID) bool {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query :=
		`select exists (select id from quote_states where id = $1)`

	var exists bool

	err := db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

func (db *DB) SetDefaultQuoteState(id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `update quote_states set is_default=false where is_default`
	_, err = tx.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	query = `update quote_states set is_default=true where id=$1`

	result, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	if affected, _ := result.RowsAffected(); affected == 0 {
		return ErrRecordNotFound
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (db *DB) GetAllQuoteStates() ([]QuoteState, f.Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
		select count(*) over(), id, value, is_default, color, is_public
		from quote_states`

	rows, err := db.QueryxContext(ctx, query)
	if err != nil {
		return nil, f.Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	var quoteStates []QuoteState

	for rows.Next() {
		var state QuoteState

		err := rows.Scan(
			&totalRecords,
			&state.ID,
			&state.Value,
			&state.IsDefault,
			&state.Color,
			&state.IsPublic,
		)

		if err != nil {
			return nil, f.Metadata{}, err
		}

		quoteStates = append(quoteStates, state)
	}

	if err = rows.Err(); err != nil {
		return nil, f.Metadata{}, err
	}

	metadata := f.CalculateMetadata(totalRecords, 1, totalRecords)

	return quoteStates, metadata, nil
}

func (db *DB) getQuoteStateById(stateID uuid.UUID) (*QuoteState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
		select id, value, is_default, color, is_public
		from quote_states
		where id=$1`

	var quoteState QuoteState

	err := db.QueryRowContext(ctx, query, stateID).Scan(
		&quoteState.ID,
		&quoteState.Value,
		&quoteState.IsDefault,
		&quoteState.Color,
		&quoteState.IsPublic,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &quoteState, nil
}
