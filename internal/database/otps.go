package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/google/uuid"
	o "javlonrahimov/quotes-api/internal/otp"
	"time"
)

type OTP struct {
	ID        uuid.UUID `db:"id"`
	Plaintext string    `db:"-"`
	Hash      string    `db:"hash"`
	UserID    uuid.UUID `db:"user_id"`
	Expiry    time.Time `db:"expiry"`
	Created   time.Time `db:"created"`
	Scope     string    `db:"scope"`
}

func (db *DB) InsertOTP(otp *OTP) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
			insert into otps (id, hash, user_id, expiry, created, "scope")
			VALUES ($1, $2, $3, $4, $5, $6)`

	args := []interface{}{otp.ID, otp.Hash, otp.UserID, otp.Expiry, otp.Created, otp.Scope}

	_, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) GetOTPForEmail(email string, scope string) (*OTP, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var otp OTP

	query := `
			select otps.id, otps.hash, otps.user_id, otps.expiry, otps.created from users
			inner join otps 
			on otps.user_id=(select users.id from users where users.email=$1)
			where otps.scope=$2
			order by created desc limit 1`

	rows, err := db.QueryxContext(ctx, query, email, scope)
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		err = rows.Scan(&otp.ID, &otp.Hash, &otp.UserID, &otp.Expiry, &otp.Created)
		if err != nil {
			return nil, err
		}
		return &otp, nil
	}
	return nil, ErrRecordNotFound
}

func (db *DB) DeleteAllOTPForUser(userID uuid.UUID, scope string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
			delete from otps
			where user_id=$1 AND "scope"=$2`

	_, err := db.ExecContext(ctx, query, userID, scope)
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) NewOtp(userID uuid.UUID, ttl time.Duration, scope string) (*OTP, error) {
	otp, err := generateOTP(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = db.InsertOTP(otp)
	return otp, err
}

func generateOTP(userID uuid.UUID, ttl time.Duration, scope string) (*OTP, error) {

	otp := &OTP{
		ID:      uuid.New(),
		UserID:  userID,
		Expiry:  time.Now().Add(ttl),
		Created: time.Now(),
		Scope:   scope,
	}

	otp.Plaintext = o.CreateOtp(4)
	hash := sha256.Sum256([]byte(otp.Plaintext))
	otp.Hash = hex.EncodeToString(hash[:])

	return otp, nil
}
