package postgres

import (
	"TelegramBot/storage"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"golang.org/x/net/html"
	"net/http"
)

type PostgresStorage struct {
	conn *pgx.Conn
}

// New initializes a new PostgresStorage instance with a database connection.
func New(conn *pgx.Conn) *PostgresStorage {
	return &PostgresStorage{
		conn: conn,
	}
}

// Save stores a page with its title in the database.
func (s *PostgresStorage) Save(p *storage.Page) error {
	// Get the user ID from the users table
	userID, err := s.getUserID(p.UserName)
	if err != nil {
		return err
	}

	// Parse the page to extract the <title>
	title, err := fetchTitle(p.URL)
	if err != nil {
		return fmt.Errorf("failed to fetch title: %v", err)
	}

	// Save the page with the title to the database
	query := `INSERT INTO pages (url, user_id, title) VALUES ($1, $2, $3)`
	_, err = s.conn.Exec(context.Background(), query, p.URL, userID, title)
	if err != nil {
		return err
	}
	return nil
}

// fetchTitle retrieves the <title> from the given URL.
func fetchTitle(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch the URL: %v", err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %v", err)
	}

	var title string
	var foundTitle bool

	// Helper function to traverse the HTML nodes and search for <title>
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" {
			if n.FirstChild != nil {
				title = n.FirstChild.Data
				foundTitle = true
			}
		}

		for c := n.FirstChild; c != nil && !foundTitle; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)

	if !foundTitle {
		return "", fmt.Errorf("no <title> tag found on the page")
	}

	return title, nil
}

// getUserID fetches the user ID associated with the given username.
func (s *PostgresStorage) getUserID(username string) (int, error) {
	query := `SELECT id FROM users WHERE username = $1`
	row := s.conn.QueryRow(context.Background(), query, username)

	var userID int
	if err := row.Scan(&userID); err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("user not found: %s", username)
		}
		return 0, err
	}
	fmt.Printf("Found user ID for %s: %d\n", username, userID) // Debug information
	return userID, nil
}

// CreateUser adds a new user to the database or returns the existing user's ID.
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

// PickRandom selects a random page for the specified user.
func (s *PostgresStorage) PickRandom(username string) (*storage.Page, error) {
	// Get the user ID
	userID, err := s.getUserID(username)
	if err != nil {
		return nil, err // Return the error if user not found
	}

	// Query for a random page for that user
	query := `SELECT url, title FROM pages WHERE user_id = $1 ORDER BY RANDOM() LIMIT 1`
	row := s.conn.QueryRow(context.Background(), query, userID)

	var p storage.Page
	p.UserName = username // Initialize UserName

	err = row.Scan(&p.URL, &p.Title) // Scan both URL and Title
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNoSavedPages // Return a specific error for no pages found
		}
		return nil, fmt.Errorf("failed to fetch random page: %v", err)
	}

	return &p, nil
}

// Remove deletes a page from the database.
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

// IsExists checks if a page exists for a given user.
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

// HasPages checks if a user has any saved pages.
func (s *PostgresStorage) HasPages(username string) (bool, error) {
	userID, err := s.getUserID(username)
	if err != nil {
		return false, err
	}

	query := `SELECT EXISTS(SELECT 1 FROM pages WHERE user_id=$1)`
	var exists bool
	err = s.conn.QueryRow(context.Background(), query, userID).Scan(&exists)
	if err != nil {
		return false, err
	}
	fmt.Printf("User %s has pages: %v\n", username, exists) // Debug information
	return exists, nil
}

// UserExists checks if a user exists in the database.
func (s *PostgresStorage) UserExists(username string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)`
	var exists bool
	err := s.conn.QueryRow(context.Background(), query, username).Scan(&exists)
	if err != nil {
		return false, err
	}
	fmt.Printf("Checking existence for user %s: %v\n", username, exists) // Debug information
	return exists, nil
}
