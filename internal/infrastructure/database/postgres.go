package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oziev02/CommentTree/internal/domain"
)

// PostgresRepository реализует CommentRepository для PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository создает новый экземпляр PostgresRepository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Create создает новый комментарий
func (r *PostgresRepository) Create(comment *domain.Comment) error {
	query := `
		INSERT INTO comments (parent_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	now := time.Now()
	comment.CreatedAt = now
	comment.UpdatedAt = now

	err := r.pool.QueryRow(
		context.Background(),
		query,
		comment.ParentID,
		comment.Content,
		comment.CreatedAt,
		comment.UpdatedAt,
	).Scan(&comment.ID)

	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	return nil
}

// GetByID получает комментарий по ID
func (r *PostgresRepository) GetByID(id int64) (*domain.Comment, error) {
	query := `
		SELECT id, parent_id, content, created_at, updated_at
		FROM comments
		WHERE id = $1
	`

	var comment domain.Comment
	var parentID sql.NullInt64

	err := r.pool.QueryRow(
		context.Background(),
		query,
		id,
	).Scan(
		&comment.ID,
		&parentID,
		&comment.Content,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, domain.ErrCommentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}

	if parentID.Valid {
		comment.ParentID = &parentID.Int64
	}

	return &comment, nil
}

// GetTree получает дерево комментариев
func (r *PostgresRepository) GetTree(parentID *int64, filter domain.CommentFilter) ([]domain.CommentTree, error) {
	var query string
	var args []interface{}

	sortBy := filter.SortBy
	if sortBy != "created_at" && sortBy != "updated_at" {
		sortBy = "created_at"
	}
	order := filter.Order
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	if parentID == nil {
		// Получаем ВСЕ комментарии (и корневые, и дочерние) для построения полного дерева
		// Затем в коде отфильтруем корневые и применим пагинацию
		query = `
			SELECT id, parent_id, content, created_at, updated_at
			FROM comments
		`
		args = []interface{}{}
	} else {
		query = fmt.Sprintf(`
			WITH RECURSIVE comment_tree AS (
				SELECT id, parent_id, content, created_at, updated_at
				FROM comments
				WHERE id = $1
				
				UNION ALL
				
				SELECT c.id, c.parent_id, c.content, c.created_at, c.updated_at
				FROM comments c
				INNER JOIN comment_tree ct ON c.parent_id = ct.id
			)
			SELECT id, parent_id, content, created_at, updated_at
			FROM comment_tree
			ORDER BY %s %s
		`, sortBy, order)
		args = []interface{}{*parentID}
	}

	rows, err := r.pool.Query(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get comment tree: %w", err)
	}
	defer rows.Close()

	comments := make(map[int64]*domain.Comment)
	var rootComments []*domain.Comment

	for rows.Next() {
		var comment domain.Comment
		var parentID sql.NullInt64

		err := rows.Scan(
			&comment.ID,
			&parentID,
			&comment.Content,
			&comment.CreatedAt,
			&comment.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}

		if parentID.Valid {
			comment.ParentID = &parentID.Int64
		}

		comments[comment.ID] = &comment

		if comment.ParentID == nil {
			rootComments = append(rootComments, &comment)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Сортируем корневые комментарии
	sortedRoots := make([]*domain.Comment, len(rootComments))
	copy(sortedRoots, rootComments)

	// Сортировка корневых комментариев
	if filter.SortBy == "created_at" {
		if filter.Order == "desc" {
			for i := 0; i < len(sortedRoots)-1; i++ {
				for j := i + 1; j < len(sortedRoots); j++ {
					if sortedRoots[i].CreatedAt.Before(sortedRoots[j].CreatedAt) {
						sortedRoots[i], sortedRoots[j] = sortedRoots[j], sortedRoots[i]
					}
				}
			}
		} else {
			for i := 0; i < len(sortedRoots)-1; i++ {
				for j := i + 1; j < len(sortedRoots); j++ {
					if sortedRoots[i].CreatedAt.After(sortedRoots[j].CreatedAt) {
						sortedRoots[i], sortedRoots[j] = sortedRoots[j], sortedRoots[i]
					}
				}
			}
		}
	} else if filter.SortBy == "updated_at" {
		if filter.Order == "desc" {
			for i := 0; i < len(sortedRoots)-1; i++ {
				for j := i + 1; j < len(sortedRoots); j++ {
					if sortedRoots[i].UpdatedAt.Before(sortedRoots[j].UpdatedAt) {
						sortedRoots[i], sortedRoots[j] = sortedRoots[j], sortedRoots[i]
					}
				}
			}
		} else {
			for i := 0; i < len(sortedRoots)-1; i++ {
				for j := i + 1; j < len(sortedRoots); j++ {
					if sortedRoots[i].UpdatedAt.After(sortedRoots[j].UpdatedAt) {
						sortedRoots[i], sortedRoots[j] = sortedRoots[j], sortedRoots[i]
					}
				}
			}
		}
	}

	// Применяем пагинацию к корневым комментариям
	start := (filter.Page - 1) * filter.PageSize
	end := start + filter.PageSize
	if start >= len(sortedRoots) {
		sortedRoots = []*domain.Comment{}
	} else if end > len(sortedRoots) {
		sortedRoots = sortedRoots[start:]
	} else {
		sortedRoots = sortedRoots[start:end]
	}

	// Строим дерево для каждого корневого комментария
	trees := make([]domain.CommentTree, 0)
	for _, root := range sortedRoots {
		tree := r.buildTree(root, comments)
		trees = append(trees, tree)
	}

	return trees, nil
}

// buildTree строит дерево комментариев рекурсивно
func (r *PostgresRepository) buildTree(comment *domain.Comment, allComments map[int64]*domain.Comment) domain.CommentTree {
	tree := domain.CommentTree{
		Comment:  *comment,
		Children: make([]domain.CommentTree, 0),
	}

	for _, c := range allComments {
		if c.ParentID != nil && *c.ParentID == comment.ID {
			childTree := r.buildTree(c, allComments)
			tree.Children = append(tree.Children, childTree)
		}
	}

	return tree
}

// Delete удаляет комментарий и все вложенные комментарии
func (r *PostgresRepository) Delete(id int64) error {
	query := `
		WITH RECURSIVE comment_tree AS (
			SELECT id
			FROM comments
			WHERE id = $1
			
			UNION ALL
			
			SELECT c.id
			FROM comments c
			INNER JOIN comment_tree ct ON c.parent_id = ct.id
		)
		DELETE FROM comments
		WHERE id IN (SELECT id FROM comment_tree)
	`

	_, err := r.pool.Exec(context.Background(), query, id)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	return nil
}

// Search выполняет полнотекстовый поиск по комментариям
func (r *PostgresRepository) Search(query string, filter domain.CommentFilter) ([]domain.CommentTree, error) {
	sortBy := filter.SortBy
	if sortBy != "created_at" && sortBy != "updated_at" {
		sortBy = "created_at"
	}
	order := filter.Order
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	searchPattern := "%" + query + "%"

	// Находим все комментарии, содержащие поисковый запрос
	searchQuery := `
		SELECT id, parent_id, content, created_at, updated_at
		FROM comments
		WHERE content ILIKE $1
	`

	searchRows, err := r.pool.Query(context.Background(), searchQuery, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search comments: %w", err)
	}
	defer searchRows.Close()

	// Собираем ID найденных комментариев и их корневых родителей
	foundCommentIDs := make(map[int64]bool)
	rootIDs := make(map[int64]bool)

	for searchRows.Next() {
		var comment domain.Comment
		var parentID sql.NullInt64

		err := searchRows.Scan(
			&comment.ID,
			&parentID,
			&comment.Content,
			&comment.CreatedAt,
			&comment.UpdatedAt,
		)
		if err != nil {
			continue
		}

		foundCommentIDs[comment.ID] = true

		// Находим корневой комментарий для каждого найденного
		rootID := comment.ID
		if parentID.Valid {
			rootID = r.findRootComment(comment.ID)
		}
		rootIDs[rootID] = true
	}

	if len(rootIDs) == 0 {
		return []domain.CommentTree{}, nil
	}

	// Получаем все комментарии для построения полного дерева
	allCommentsQuery := `SELECT id, parent_id, content, created_at, updated_at FROM comments`
	allRows, err := r.pool.Query(context.Background(), allCommentsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get all comments: %w", err)
	}
	defer allRows.Close()

	allComments := make(map[int64]*domain.Comment)
	var rootComments []*domain.Comment

	for allRows.Next() {
		var comment domain.Comment
		var parentID sql.NullInt64

		err := allRows.Scan(
			&comment.ID,
			&parentID,
			&comment.Content,
			&comment.CreatedAt,
			&comment.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if parentID.Valid {
			comment.ParentID = &parentID.Int64
		}

		allComments[comment.ID] = &comment

		// Добавляем только корневые комментарии, которые есть в результатах поиска
		if comment.ParentID == nil && rootIDs[comment.ID] {
			rootComments = append(rootComments, &comment)
		}
	}

	// Сортируем корневые комментарии
	sortedRoots := make([]*domain.Comment, len(rootComments))
	copy(sortedRoots, rootComments)

	if sortBy == "created_at" {
		if order == "desc" {
			for i := 0; i < len(sortedRoots)-1; i++ {
				for j := i + 1; j < len(sortedRoots); j++ {
					if sortedRoots[i].CreatedAt.Before(sortedRoots[j].CreatedAt) {
						sortedRoots[i], sortedRoots[j] = sortedRoots[j], sortedRoots[i]
					}
				}
			}
		} else {
			for i := 0; i < len(sortedRoots)-1; i++ {
				for j := i + 1; j < len(sortedRoots); j++ {
					if sortedRoots[i].CreatedAt.After(sortedRoots[j].CreatedAt) {
						sortedRoots[i], sortedRoots[j] = sortedRoots[j], sortedRoots[i]
					}
				}
			}
		}
	} else if sortBy == "updated_at" {
		if order == "desc" {
			for i := 0; i < len(sortedRoots)-1; i++ {
				for j := i + 1; j < len(sortedRoots); j++ {
					if sortedRoots[i].UpdatedAt.Before(sortedRoots[j].UpdatedAt) {
						sortedRoots[i], sortedRoots[j] = sortedRoots[j], sortedRoots[i]
					}
				}
			}
		} else {
			for i := 0; i < len(sortedRoots)-1; i++ {
				for j := i + 1; j < len(sortedRoots); j++ {
					if sortedRoots[i].UpdatedAt.After(sortedRoots[j].UpdatedAt) {
						sortedRoots[i], sortedRoots[j] = sortedRoots[j], sortedRoots[i]
					}
				}
			}
		}
	}

	// Применяем пагинацию
	start := (filter.Page - 1) * filter.PageSize
	end := start + filter.PageSize
	if start >= len(sortedRoots) {
		sortedRoots = []*domain.Comment{}
	} else if end > len(sortedRoots) {
		sortedRoots = sortedRoots[start:]
	} else {
		sortedRoots = sortedRoots[start:end]
	}

	// Строим дерево для каждого корневого комментария
	trees := make([]domain.CommentTree, 0)
	for _, root := range sortedRoots {
		fullTree := r.buildTree(root, allComments)
		trees = append(trees, fullTree)
	}

	return trees, nil
}

// findRootComment находит корневой комментарий для данного комментария
func (r *PostgresRepository) findRootComment(commentID int64) int64 {
	query := `
		WITH RECURSIVE comment_path AS (
			SELECT id, parent_id
			FROM comments
			WHERE id = $1
			
			UNION ALL
			
			SELECT c.id, c.parent_id
			FROM comments c
			INNER JOIN comment_path cp ON c.id = cp.parent_id
		)
		SELECT id FROM comment_path WHERE parent_id IS NULL LIMIT 1
	`

	var rootID int64
	err := r.pool.QueryRow(context.Background(), query, commentID).Scan(&rootID)
	if err != nil {
		// Если не удалось найти корневой, возвращаем сам ID
		return commentID
	}

	return rootID
}

// getFullTree получает полное дерево комментария
func (r *PostgresRepository) getFullTree(rootID int64) domain.CommentTree {
	query := `
		WITH RECURSIVE comment_tree AS (
			SELECT id, parent_id, content, created_at, updated_at
			FROM comments
			WHERE id = $1
			
			UNION ALL
			
			SELECT c.id, c.parent_id, c.content, c.created_at, c.updated_at
			FROM comments c
			INNER JOIN comment_tree ct ON c.parent_id = ct.id
		)
		SELECT id, parent_id, content, created_at, updated_at
		FROM comment_tree
	`

	rows, err := r.pool.Query(context.Background(), query, rootID)
	if err != nil {
		// Если ошибка, возвращаем только корневой комментарий
		comment, _ := r.GetByID(rootID)
		if comment != nil {
			return domain.CommentTree{Comment: *comment}
		}
		return domain.CommentTree{}
	}
	defer rows.Close()

	comments := make(map[int64]*domain.Comment)
	var rootComment *domain.Comment

	for rows.Next() {
		var comment domain.Comment
		var parentID sql.NullInt64

		err := rows.Scan(
			&comment.ID,
			&parentID,
			&comment.Content,
			&comment.CreatedAt,
			&comment.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if parentID.Valid {
			comment.ParentID = &parentID.Int64
		}

		comments[comment.ID] = &comment

		if comment.ParentID == nil {
			rootComment = &comment
		}
	}

	if rootComment == nil {
		return domain.CommentTree{}
	}

	return r.buildTree(rootComment, comments)
}

// Count возвращает количество комментариев
func (r *PostgresRepository) Count(parentID *int64, search string) (int, error) {
	var query string
	var args []interface{}

	if search != "" {
		query = `
			SELECT COUNT(DISTINCT id)
			FROM comments
			WHERE content ILIKE $1
		`
		args = []interface{}{"%" + search + "%"}
	} else if parentID == nil {
		query = `
			SELECT COUNT(*)
			FROM comments
			WHERE parent_id IS NULL
		`
		args = []interface{}{}
	} else {
		query = `
			WITH RECURSIVE comment_tree AS (
				SELECT id
				FROM comments
				WHERE id = $1
				
				UNION ALL
				
				SELECT c.id
				FROM comments c
				INNER JOIN comment_tree ct ON c.parent_id = ct.id
			)
			SELECT COUNT(*)
			FROM comment_tree
		`
		args = []interface{}{*parentID}
	}

	var count int
	err := r.pool.QueryRow(context.Background(), query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count comments: %w", err)
	}

	return count, nil
}
