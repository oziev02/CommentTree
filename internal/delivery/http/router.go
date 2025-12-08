package http

import (
	"net/http"

	"github.com/oziev02/CommentTree/internal/usecase"
)

// NewRouter создает HTTP роутер
func NewRouter(commentUseCase *usecase.CommentUseCase) *http.ServeMux {
	handler := NewCommentHandler(commentUseCase)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /comments", handler.Create)
	mux.HandleFunc("GET /comments", handler.GetTree)
	mux.HandleFunc("DELETE /comments/{id}", handler.Delete)

	return mux
}
