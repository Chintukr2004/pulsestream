package generator

import (
	"math/rand"
	"time"

	"github.com/Chintukr2004/pulsestream/internal/model"
	"github.com/google/uuid"
)

var usernames = []string{
	"alice", "bob", "charlie", "dhrona", "gopher",
}

var contents = []string{
	"I love Golang concurrency!",
	"This backend is blazing fast!",
	"Learning distributed systems!",
	"Microservices are powerful!",
	"Channels are awesome!",
}

func GeneratePost() model.Post {
	return model.Post{
		ID:        uuid.New().String(),
		Username:  usernames[rand.Intn(len(usernames))],
		Content:   contents[rand.Intn(len(contents))],
		CreatedAt: time.Now(),
	}
}
