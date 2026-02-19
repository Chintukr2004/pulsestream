package store

import (
	"context"

	"github.com/Chintukr2004/pulsestream/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostStore struct {
	DB *pgxpool.Pool
}

func NewPostStore(db *pgxpool.Pool) *PostStore {
	return &PostStore{DB: db}
}

func (s *PostStore) Insert(ctx context.Context, post model.Post) error {
	query := `
		INSERT INTO posts(id, username, content, created_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err := s.DB.Exec(ctx, query, post.ID, post.Username, post.Content, post.CreatedAt)
	return err

}

func (s *PostStore) GetLatest(ctx context.Context, limit int) ([]model.Post, error) {
	query := `
	SELECT id, username, content, created_at
	FROM posts
	ORDER BY created_at DESC
	LIMIT $1
	`
	rows, err := s.DB.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []model.Post

	for rows.Next() {
		var p model.Post
		err := rows.Scan(
			&p.ID,
			&p.Username,
			&p.Content,
			&p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, nil
}
