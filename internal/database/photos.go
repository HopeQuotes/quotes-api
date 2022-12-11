package database

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	f "javlonrahimov/quotes-api/internal/filters"
)

type Photo struct {
	ID       uuid.UUID `db:"id"`
	Color    string    `db:"color"`
	BlurHash string    `db:"blur_hash"`
	Author   string    `db:"author"`
	Url      string    `db:"url"`
}

func (db *DB) InsertPhoto(color, blurHash, author, url string) (*Photo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	photo := Photo{
		ID:       uuid.New(),
		Color:    color,
		BlurHash: blurHash,
		Author:   author,
		Url:      url,
	}

	query := `
 		insert into photos(id, color, blur_hash, author, url)
 		values ($1, $2, $3, $4, $5)`

	args := []interface{}{photo.ID, photo.Color, photo.BlurHash, photo.Author, photo.Url}

	_, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return &photo, nil
}

func (db *DB) GetPhotoById(id uuid.UUID) (*Photo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
		select id, color, blur_hash, author, url
		from photos where id = $1`

	var photo Photo

	err := db.QueryRowContext(ctx, query, id).Scan(&photo.ID, &photo.Color, &photo.BlurHash, &photo.Author, &photo.Url)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &photo, nil
}

func (db *DB) GetPhotos() ([]*Photo, f.Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
		select count(*) over(), id, color, blur_hash, author, url
		from photos`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, f.Metadata{}, err
	}

	defer rows.Close()

	totalRecords := 0
	photos := []*Photo{}

	for rows.Next() {
		var photo Photo

		err := rows.Scan(
			&totalRecords,
			&photo.ID,
			&photo.Color,
			&photo.BlurHash,
			&photo.Author,
			&photo.Url,
		)

		if err != nil {
			return nil, f.Metadata{}, err
		}

		photos = append(photos, &photo)
	}

	if err = rows.Err(); err != nil {
		return nil, f.Metadata{}, err
	}

	metadata := f.CalculateMetadata(totalRecords, 1, totalRecords)

	return photos, metadata, nil

}
