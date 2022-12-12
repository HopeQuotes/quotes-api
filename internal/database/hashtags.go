package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	f "javlonrahimov/quotes-api/internal/filters"
)

type Hashtag struct {
	ID    uuid.UUID `db:"id"`
	Value string    `db:"value"`
}

func (db *DB) InsertHashtag(value string) (*Hashtag, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	hashtag := Hashtag{
		ID:    uuid.New(),
		Value: value,
	}

	query := `
		insert into hashtags (id, value)
		values ($1, $2)`

	_, err := db.ExecContext(ctx, query, hashtag.ID, hashtag.Value)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "hashtags_value_key"`:
			return nil, ErrDuplicateHashtag
		default:
			return nil, err
		}
	}
	return &hashtag, nil
}

func (db *DB) DeleteHashtagById(id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `delete from hashtags where id = $1`

	_, err := db.ExecContext(ctx, query, id)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrRecordNotFound
		default:
			return err
		}
	}

	return nil
}

func (db *DB) IsQuoteExistsWithThisHashtag(hashtagID uuid.UUID) bool {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query :=
		`select exists (select id from quote_hashtags where hashtag_id = $1)`

	var exists bool

	err := db.QueryRowContext(ctx, query, hashtagID).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

func (db *DB) GetQuoteHashtags(quoteID uuid.UUID) ([]*Hashtag, f.Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
		select count(*) over(), h.id, h.value
		from hashtags h
		inner join quote_hashtags q
		on h.id = q.hashtag_id
		where q.quote_id = $1`

	rows, err := db.QueryContext(ctx, query, quoteID)
	if err != nil {
		return nil, f.Metadata{}, err
	}

	totalRecords := 0
	hashtags := []*Hashtag{}

	for rows.Next() {
		var hashtag Hashtag

		err := rows.Scan(
			&totalRecords, &hashtag.ID, &hashtag.Value,
		)

		if err != nil {
			return nil, f.Metadata{}, err
		}

		hashtags = append(hashtags, &hashtag)
	}
	if err := rows.Err(); err != nil {
		return nil, f.Metadata{}, err
	}

	metadata := f.CalculateMetadata(totalRecords, 1, totalRecords)

	return hashtags, metadata, nil
}

func (db *DB) GetHashtags(filters f.Filters) ([]*Hashtag, f.Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := fmt.Sprintf(`
		select count(*) over(), id, value
		from hashtags
		order by %s %s, value asc
		limit $1 offset $2`, filters.SortColumn(), filters.SortDirection())

	rows, err := db.QueryContext(ctx, query, filters.Limit(), filters.Offset())
	if err != nil {
		return nil, f.Metadata{}, err
	}

	totalRecords := 0
	hashtags := []*Hashtag{}

	for rows.Next() {
		var hashtag Hashtag

		err := rows.Scan(
			&totalRecords, &hashtag.ID, &hashtag.Value,
		)

		if err != nil {
			return nil, f.Metadata{}, err
		}

		hashtags = append(hashtags, &hashtag)
	}
	if err := rows.Err(); err != nil {
		return nil, f.Metadata{}, err
	}

	metadata := f.CalculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return hashtags, metadata, nil
}
