package database

import (
	"context"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

func (db *DB) GetAllPermissionsForUser(userID uuid.UUID) ([]string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
		SELECT permissions.code
		FROM permissions
		INNER JOIN users_permissions ON users_permissions.permission_id = permissions.id
		INNER JOIN users ON users_permissions.user_id = users.id
		WHERE users.id = $1`

	rows, err := db.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []string

	for rows.Next() {
		var permission string

		err := rows.Scan(&permission)
		if err != nil {
			return nil, err
		}

		permissions = append(permissions, permission)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}

func (db *DB) AddPermissionForUser(userID uuid.UUID, codes ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query := `
	insert into users_permissions
	select $1, permissions.id from permissions where permissions.code = any($2)`

	_, err := db.DB.ExecContext(ctx, query, userID, pq.Array(codes))

	return err
}
