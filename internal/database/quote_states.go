package database

import (
	"context"
	"github.com/google/uuid"
	f "javlonrahimov/quotes-api/internal/filters"
)

type QuoteState struct {
	ID        uuid.UUID `db:"id"`
	Value     string    `db:"value"`
	IsDefault bool      `db:"is_default"`
	Color     string    `db:"color"`
}

func (db *DB) InsertQuoteState(state *QuoteState) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
		insert into quote_states (id, value, is_default, color)
		values ($1, $2, $3, $4)
		returning id`

	args := []interface{}{uuid.New(), state.Value, false, state.Color}

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
 		select id, value, is_default, color
		from quote_states
		where is_default = true`

	err := db.QueryRowContext(ctx, query).Scan(&state.ID, &state.Value, &state.IsDefault, &state.Color)
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
		`select id from quote_states where state = $1`

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

func (db *DB) GetAllQuoteStates() ([]*QuoteState, f.Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
		select count(*) over(), id, value, is_default, color
		from quote_states`

	rows, err := db.QueryxContext(ctx, query)
	if err != nil {
		return nil, f.Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	quoteStates := []*QuoteState{}

	for rows.Next() {
		var state QuoteState

		err := rows.Scan(
			&totalRecords,
			&state.ID,
			&state.Value,
			&state.IsDefault,
			&state.Color,
		)

		if err != nil {
			return nil, f.Metadata{}, err
		}

		quoteStates = append(quoteStates, &state)
	}

	if err = rows.Err(); err != nil {
		return nil, f.Metadata{}, err
	}

	metadata := f.CalculateMetadata(totalRecords, 1, totalRecords)

	return quoteStates, metadata, nil
}
