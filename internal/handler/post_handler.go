package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Chintukr2004/pulsestream/internal/store"
)

type PostHandler struct {
	Store *store.PostStore
}

func NewPostHandler(store *store.PostStore) *PostHandler {
	return &PostHandler{Store: store}
}

func (h *PostHandler) GetPosts(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	posts, err := h.Store.GetLatest(ctx, 20)
	if err != nil {
		http.Error(w, "Failed to fetch posts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}
