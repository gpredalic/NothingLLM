package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("not found")

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func (d *DB) CreateUser(ctx context.Context, email, password, role string) (*User, error) {
	u := &User{}
	err := d.pool.QueryRow(ctx, `
		INSERT INTO users(email, password, role) VALUES($1,$2,$3)
		RETURNING id, email, password, role, created_at`,
		email, password, role,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (d *DB) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := d.pool.QueryRow(ctx, `
		SELECT id, email, password, role, created_at FROM users WHERE email=$1`, email,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (d *DB) GetUserByID(ctx context.Context, id string) (*User, error) {
	u := &User{}
	err := d.pool.QueryRow(ctx, `
		SELECT id, email, password, role, created_at FROM users WHERE id=$1`, id,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (d *DB) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := d.pool.Query(ctx, `SELECT id, email, role, created_at FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (d *DB) UpdateUserRole(ctx context.Context, id, role string) error {
	tag, err := d.pool.Exec(ctx, `UPDATE users SET role=$1, updated_at=now() WHERE id=$2`, role, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *DB) UpdateUserPassword(ctx context.Context, id, password string) error {
	_, err := d.pool.Exec(ctx, `UPDATE users SET password=$1, updated_at=now() WHERE id=$2`, password, id)
	return err
}

func (d *DB) DeleteUser(ctx context.Context, id string) error {
	tag, err := d.pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}