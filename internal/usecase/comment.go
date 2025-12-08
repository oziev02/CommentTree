package usecase

import (
	"context"
	"fmt"

	"github.com/oziev02/CommentTree/internal/domain"
)

// CommentUseCase содержит бизнес-логику для работы с комментариями
type CommentUseCase struct {
	repo domain.CommentRepository
}

// NewCommentUseCase создает новый экземпляр CommentUseCase
func NewCommentUseCase(repo domain.CommentRepository) *CommentUseCase {
	return &CommentUseCase{repo: repo}
}

// Create создает новый комментарий
func (uc *CommentUseCase) Create(ctx context.Context, parentID *int64, content string) (*domain.Comment, error) {
	if content == "" {
		return nil, domain.ErrEmptyContent
	}

	comment := &domain.Comment{
		ParentID: parentID,
		Content:  content,
	}

	if parentID != nil {
		parent, err := uc.repo.GetByID(*parentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent comment: %w", err)
		}
		if parent == nil {
			return nil, domain.ErrInvalidParent
		}
	}

	if err := uc.repo.Create(comment); err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	return comment, nil
}

// GetTree получает дерево комментариев
func (uc *CommentUseCase) GetTree(ctx context.Context, filter domain.CommentFilter) ([]domain.CommentTree, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 50
	}
	if filter.SortBy == "" {
		filter.SortBy = "created_at"
	}
	if filter.Order == "" {
		filter.Order = "desc"
	}

	if filter.Search != "" {
		return uc.repo.Search(filter.Search, filter)
	}

	return uc.repo.GetTree(filter.ParentID, filter)
}

// Delete удаляет комментарий и все вложенные комментарии
func (uc *CommentUseCase) Delete(ctx context.Context, id int64) error {
	_, err := uc.repo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get comment: %w", err)
	}

	if err := uc.repo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	return nil
}

// GetTotalCount возвращает общее количество комментариев
func (uc *CommentUseCase) GetTotalCount(ctx context.Context, parentID *int64, search string) (int, error) {
	return uc.repo.Count(parentID, search)
}
