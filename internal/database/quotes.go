package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	f "javlonrahimov/quotes-api/internal/filters"
	"time"

	"github.com/google/uuid"
)

type Quote struct {
	ID        uuid.UUID  `db:"id"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	CreatedBy uuid.UUID  `db:"created_by"`
	Author    string     `db:"author"`
	Text      string     `db:"text"`
	State     QuoteState `db:"-"`
	Photo     *Photo     `db:"-"`
	Hashtags  []Hashtag  `db:"-"`
}

func (db *DB) InsertQuote(author, text string, userID, photoID uuid.UUID, hashtagIDs []uuid.UUID, stateID *uuid.UUID) (*Quote, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	state, err := db.getDefaultQuoteState()
	if err != nil && state != nil {
		return nil, err
	}

	if stateID != nil {
		state, err = db.getQuoteStateById(*stateID)
		if err != nil {
			switch {
			case errors.Is(err, sql.ErrNoRows):
				return nil, ErrRecordNotFound
			default:
				return nil, err
			}
		}
	}

	photo, err := db.GetPhotoById(photoID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	quote := &Quote{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Author:    author,
		Text:      text,
		CreatedBy: userID,
		State:     *state,
		Photo:     photo,
	}

	// todo change state to state_id
	query := `
		insert into quotes (id, created_at, updated_at, author, text, created_by, state, photo_id)
		values ($1, $2, $3, $4, $5, $6, $7, $8)`

	args := []interface{}{quote.ID, quote.CreatedAt, quote.UpdatedAt, quote.Author, quote.Text, quote.CreatedBy, state.ID, photo.ID}

	_, err = db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	err = db.insertQuoteHashtags(hashtagIDs, quote.ID)
	if err != nil {
		return nil, err
	}

	hashtags, _, err := db.GetQuoteHashtags(quote.ID)
	if err != nil {
		return nil, err
	}

	quote.Hashtags = hashtags

	return quote, nil
}

func (db *DB) UpdateQuote(quoteID, photoID uuid.UUID, author, text string, hashtagIDs []uuid.UUID) (*Quote, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	quote, err := db.GetQuoteById(quoteID)
	if err != nil {
		return nil, err
	}

	quote.Author = author
	quote.Text = text

	err = db.insertQuoteHashtags(hashtagIDs, quote.ID)
	if err != nil {
		return nil, err
	}

	hashtags, _, err := db.GetQuoteHashtags(quote.ID)
	if err != nil {
		return nil, err
	}

	quote.Hashtags = hashtags

	photo, err := db.GetPhotoById(photoID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	quote.Photo = photo

	query := `
		update quotes
		set text = $1, author = $2, photo_id = $3, updated_at = $4
		where id = $5
		returning updated_at`

	err = db.QueryRowContext(ctx, query, quote.Text, photo.ID, quote.Author, time.Now()).Scan(&quote.UpdatedAt)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return quote, nil
}

func (db *DB) DeleteQuoteById(id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `delete from quotes where id = $1`

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

func (db *DB) GetQuoteById(id uuid.UUID) (*Quote, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
		select q.id, q.created_at, q.updated_at, q.created_by, q.author, q.text, s.value, p.id, p.url, p.color, p.blur_hash, p.author
		from quotes q
		inner join quote_states s
		on q.state = s.id
		inner join photos p 
		on q.photo_id = p.id
		where id = $1`

	var quote Quote

	err := db.QueryRowContext(ctx, query, id).Scan(
		&quote.ID, &quote.CreatedAt, &quote.UpdatedAt,
		&quote.CreatedBy, &quote.Author, &quote.Text,
		&quote.State, &quote.Photo.ID, &quote.Photo.Url,
		&quote.Photo.Color, &quote.Photo.BlurHash, &quote.Photo.Author,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	hashtags, _, err := db.GetQuoteHashtags(id)
	if err != nil {
		return nil, err
	}

	quote.Hashtags = hashtags

	return &quote, nil
}

func (db *DB) GetUserQuotes(userID uuid.UUID, author string, text string, state uuid.UUID, filters f.Filters) ([]Quote, f.Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := fmt.Sprintf(`
		select count(*) over(), q.id, q.created_at, q.updated_at, q.created_by, q.author, text, s.id, s.value, s.is_default, s.color, s.is_public, p.id, p.url, p.color, p.blur_hash, p.author
		from quotes q
		inner join quote_states s
		on q.state = s.id
		inner join photos p 
		on q.photo_id = p.id
		where (to_tsvector('simple', q.author) @@ plainto_tsquery('simple', $1) or $1 = '')
		and (to_tsvector('simple', q.text) @@ plainto_tsquery('simple', $2) or $2 = '')
		and q.created_by = $3
		and s.id = $4
		order by q.%s %s, q.created_at asc
		limit $5 offset $6`, filters.SortColumn(), filters.SortDirection())

	args := []interface{}{author, text, userID, state, filters.Limit(), filters.Offset()}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, f.Metadata{}, err
	}

	defer rows.Close()

	totalRecords := 0
	quotes := make([]Quote, 0)

	for rows.Next() {
		quote := Quote{Photo: &Photo{}}

		err := rows.Scan(
			&totalRecords,
			&quote.ID,
			&quote.CreatedAt,
			&quote.UpdatedAt,
			&quote.CreatedBy,
			&quote.Author,
			&quote.Text,
			&quote.State.ID,
			&quote.State.Value,
			&quote.State.IsDefault,
			&quote.State.Color,
			&quote.State.IsPublic,
			&quote.Photo.ID,
			&quote.Photo.Url,
			&quote.Photo.Color,
			&quote.Photo.BlurHash,
			&quote.Photo.Author,
		)
		if err != nil {
			return nil, f.Metadata{}, err
		}

		hashtags, _, err := db.GetQuoteHashtags(quote.ID)
		if err != nil {
			return nil, f.Metadata{}, err
		}

		quote.Hashtags = hashtags

		quotes = append(quotes, quote)
	}

	if err = rows.Err(); err != nil {
		return nil, f.Metadata{}, err
	}

	metadata := f.CalculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return quotes, metadata, nil
}

func (db *DB) GetQuotes(author string, text string, filters f.Filters) ([]Quote, f.Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
		select count(*) over(), q.id, q.created_at, q.updated_at, q.created_by, q.author, text, s.id, s.value, s.is_default, s.color, s.is_public, p.id, p.url, p.color, p.blur_hash, p.author
		from quotes q
		inner join quote_states s
		on q.state = s.id and s.is_public=true
		inner join photos p 
		on q.photo_id = p.id
		where (to_tsvector('simple', q.author) @@ plainto_tsquery('simple', $1) or $1 = '')
		and (to_tsvector('simple', q.text) @@ plainto_tsquery('simple', $2) or $2 = '')
		order by case when $3 = 'text' then q.text end
		         , case when $3 = 'created_at' then q.created_at end
		limit $4 offset $5`
	// q.%s %s, q.created_at as
	// filters.SortColumn(), filters.SortDirection())

	args := []interface{}{author, text, filters.SortColumn(), filters.Limit(), filters.Offset()}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, f.Metadata{}, err
	}

	defer rows.Close()

	totalRecords := 0
	quotes := make([]Quote, 0)

	for rows.Next() {
		quote := Quote{Photo: &Photo{}}

		err := rows.Scan(
			&totalRecords,
			&quote.ID,
			&quote.CreatedAt,
			&quote.UpdatedAt,
			&quote.CreatedBy,
			&quote.Author,
			&quote.Text,
			&quote.State.ID,
			&quote.State.Value,
			&quote.State.IsDefault,
			&quote.State.Color,
			&quote.State.IsPublic,
			&quote.Photo.ID,
			&quote.Photo.Url,
			&quote.Photo.Color,
			&quote.Photo.BlurHash,
			&quote.Photo.Author,
		)
		if err != nil {
			return nil, f.Metadata{}, err
		}

		hashtags, _, err := db.GetQuoteHashtags(quote.ID)
		if err != nil {
			return nil, f.Metadata{}, err
		}

		quote.Hashtags = hashtags

		quotes = append(quotes, quote)
	}

	if err = rows.Err(); err != nil {
		return nil, f.Metadata{}, err
	}

	metadata := f.CalculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return quotes, metadata, nil
}

func (db *DB) SetQuoteState(id, stateID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	if !db.ExistsQuoteStateById(stateID) {
		return ErrRecordNotFound
	}

	query := `update quotes
		set state = $1, updated_at = $2
		where id = $3
		returning updated_at`

	result, err := db.ExecContext(ctx, query, stateID, time.Now(), id)
	if err != nil {
		return err
	}

	if affected, _ := result.RowsAffected(); affected < 1 {
		return ErrRecordNotFound
	}

	return nil
}

func (db *DB) IsExistsWithThisState(quoteStateID uuid.UUID) bool {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query :=
		`select exists (select id from quotes where state = $1)`

	var exists bool

	err := db.QueryRowContext(ctx, query, quoteStateID).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

func (db *DB) insertQuoteHashtags(hashtagIDs []uuid.UUID, quoteID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `insert into quote_hashtags (quote_id, hashtag_id)
		values ($1, $2)`

	for _, id := range hashtagIDs {
		_, err := db.ExecContext(ctx, query, quoteID, id)
		if err != nil {
			return err
		}
	}

	return nil
}
