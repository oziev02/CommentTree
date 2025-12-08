package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/oziev02/CommentTree/internal/domain"
	"github.com/oziev02/CommentTree/internal/usecase"
)

// CommentHandler обрабатывает HTTP запросы для комментариев
type CommentHandler struct {
	useCase *usecase.CommentUseCase
}

// NewCommentHandler создает новый экземпляр CommentHandler
func NewCommentHandler(useCase *usecase.CommentUseCase) *CommentHandler {
	return &CommentHandler{useCase: useCase}
}

// CreateCommentRequest DTO для создания комментария
type CreateCommentRequest struct {
	ParentID *int64 `json:"parent_id"`
	Content  string `json:"content"`
}

// CommentResponse DTO для ответа с комментарием
type CommentResponse struct {
	ID        int64  `json:"id"`
	ParentID  *int64 `json:"parent_id,omitempty"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CommentTreeResponse DTO для ответа с деревом комментариев
type CommentTreeResponse struct {
	Comment  CommentResponse       `json:"comment"`
	Children []CommentTreeResponse `json:"children,omitempty"`
}

// CommentsListResponse DTO для списка комментариев с пагинацией
type CommentsListResponse struct {
	Comments []CommentTreeResponse `json:"comments"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

// Create обрабатывает POST /comments
func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	comment, err := h.useCase.Create(r.Context(), req.ParentID, req.Content)
	if err != nil {
		switch err {
		case domain.ErrEmptyContent:
			http.Error(w, err.Error(), http.StatusBadRequest)
		case domain.ErrInvalidParent:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toCommentResponse(comment))
}

// GetTree обрабатывает GET /comments
func (h *CommentHandler) GetTree(w http.ResponseWriter, r *http.Request) {
	filter := domain.CommentFilter{}

	if parentIDStr := r.URL.Query().Get("parent"); parentIDStr != "" {
		parentID, err := strconv.ParseInt(parentIDStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid parent_id", http.StatusBadRequest)
			return
		}
		filter.ParentID = &parentID
	}

	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = search
	}

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err == nil && page > 0 {
			filter.Page = page
		}
	}

	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err == nil && pageSize > 0 {
			filter.PageSize = pageSize
		}
	}

	if sortBy := r.URL.Query().Get("sort_by"); sortBy != "" {
		if sortBy == "created_at" || sortBy == "updated_at" {
			filter.SortBy = sortBy
		}
	}

	if order := r.URL.Query().Get("order"); order != "" {
		if order == "asc" || order == "desc" {
			filter.Order = order
		}
	}

	trees, err := h.useCase.GetTree(r.Context(), filter)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	total, err := h.useCase.GetTotalCount(r.Context(), filter.ParentID, filter.Search)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	response := CommentsListResponse{
		Comments: toCommentTreeResponseList(trees),
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Delete обрабатывает DELETE /comments/{id}
func (h *CommentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid comment id", http.StatusBadRequest)
		return
	}

	if err := h.useCase.Delete(r.Context(), id); err != nil {
		switch err {
		case domain.ErrCommentNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// toCommentResponse преобразует domain.Comment в CommentResponse
func toCommentResponse(c *domain.Comment) CommentResponse {
	return CommentResponse{
		ID:        c.ID,
		ParentID:  c.ParentID,
		Content:   c.Content,
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// toCommentTreeResponse преобразует domain.CommentTree в CommentTreeResponse
func toCommentTreeResponse(tree domain.CommentTree) CommentTreeResponse {
	response := CommentTreeResponse{
		Comment:  toCommentResponse(&tree.Comment),
		Children: make([]CommentTreeResponse, 0, len(tree.Children)),
	}

	for _, child := range tree.Children {
		response.Children = append(response.Children, toCommentTreeResponse(child))
	}

	return response
}

// toCommentTreeResponseList преобразует список domain.CommentTree в список CommentTreeResponse
func toCommentTreeResponseList(trees []domain.CommentTree) []CommentTreeResponse {
	responses := make([]CommentTreeResponse, 0, len(trees))
	for _, tree := range trees {
		responses = append(responses, toCommentTreeResponse(tree))
	}
	return responses
}
