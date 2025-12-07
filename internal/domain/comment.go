package domain

import (
	"time"
)

// Comment представляет комментарий в дереве
type Comment struct {
	ID        int64     `json:"id"`
	ParentID  *int64    `json:"parent_id,omitempty"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CommentTree представляет комментарий со всеми вложенными комментариями
type CommentTree struct {
	Comment  Comment       `json:"comment"`
	Children []CommentTree `json:"children,omitempty"`
}

// CommentFilter содержит параметры фильтрации и пагинации
type CommentFilter struct {
	ParentID *int64
	Search   string
	Page     int
	PageSize int
	SortBy   string // "created_at", "updated_at"
	Order    string // "asc", "desc"
}

// CommentRepository определяет интерфейс для работы с комментариями
type CommentRepository interface {
	Create(comment *Comment) error
	GetByID(id int64) (*Comment, error)
	GetTree(parentID *int64, filter CommentFilter) ([]CommentTree, error)
	Delete(id int64) error
	Search(query string, filter CommentFilter) ([]CommentTree, error)
	Count(parentID *int64, search string) (int, error)
}
