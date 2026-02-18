package model

import "time"

type Post struct{
	ID string
	Username string
	Content string
	CreatedAt time.Time
}