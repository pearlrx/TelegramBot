package postgres

import (
	"TelegramBot/storage"
	"context"
	"errors"
	"github.com/jackc/pgx/v4"
)

type PostgresStorage struct {
	conn *pgx.Conn
}

func New(conn *pgx.Conn) *PostgresStorage {
	return &PostgresStorage{
		conn: conn,
	}
}

func (s *PostgresStorage) Save(p *storage.Page) error {
	// Получаем ID пользователя из таблицы users
	userID, err := s.getUserID(p.UserName)
	if err != nil {
		return err
	}

	query := `INSERT INTO pages (url, user_id) VALUES ($1, $2)`
	_, err = s.conn.Exec(context.Background(), query, p.URL, userID)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStorage) getUserID(username string) (int, error) {
	query := `SELECT id FROM users WHERE username=$1`
	var userID int
	err := s.conn.QueryRow(context.Background(), query, username).Scan(&userID)
	if err == pgx.ErrNoRows {
		// Return error if no user found
		return 0, pgx.ErrNoRows
	} else if err != nil {
		return 0, err
	}
	return userID, nil
}

func (s *PostgresStorage) CreateUser(username string) (int, error) {
	// Check if the user already exists before trying to insert
	userID, err := s.getUserID(username)
	if err == nil {
		// User already exists, return the existing user ID
		return userID, nil
	} else if err != pgx.ErrNoRows {
		// Some other error occurred
		return 0, err
	}

	// User doesn't exist, so insert a new user
	query := `INSERT INTO users (username) VALUES ($1) RETURNING id`
	var newUserID int
	err = s.conn.QueryRow(context.Background(), query, username).Scan(&newUserID)
	if err != nil {
		return 0, err
	}

	return newUserID, nil
}

func (s *PostgresStorage) PickRandom(username string) (*storage.Page, error) {
	userID, err := s.getUserID(username)
	if err != nil {
		return nil, err
	}

	query := `SELECT url FROM pages WHERE user_id=$1 ORDER BY random() LIMIT 1`
	row := s.conn.QueryRow(context.Background(), query, userID)

	var page storage.Page
	page.UserName = username
	if err := row.Scan(&page.URL); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNoSavedPages
		}
		return nil, err
	}

	return &page, nil
}

func (s *PostgresStorage) Remove(p *storage.Page) error {
	userID, err := s.getUserID(p.UserName)
	if err != nil {
		return err
	}

	query := `DELETE FROM pages WHERE url=$1 AND user_id=$2`
	_, err = s.conn.Exec(context.Background(), query, p.URL, userID)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStorage) IsExists(p *storage.Page) (bool, error) {
	userID, err := s.getUserID(p.UserName)
	if err != nil {
		return false, err
	}

	query := `SELECT EXISTS(SELECT 1 FROM pages WHERE url=$1 AND user_id=$2)`
	var exists bool
	err = s.conn.QueryRow(context.Background(), query, p.URL, userID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (s *PostgresStorage) UserExists(username string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)`
	var exists bool
	err := s.conn.QueryRow(context.Background(), query, username).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
