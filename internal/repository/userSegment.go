package repository

import (
	"context"
	"database/sql"
	"fmt"
	"progression1/internal/apperror"
	"progression1/internal/model"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type SegmentRepo interface {
	CreateSegment(ctx context.Context, slug string, auto_percent *int) error
	DeleteSegment(ctx context.Context, slug string) error

	GetAllSegments(ctx context.Context) ([]string, error)
	GetHTable(ctx context.Context) ([]model.HistoryTableDTO, error)
	GetHForPeriod(ctx context.Context, year, month int) ([]model.HistoryTableDTO, error)
	SegmentExists(ctx context.Context, slug string) (bool, error)
	AddUserToSegment(ctx context.Context, userID int64, slug string) error
	UpdateUserSegments(ctx context.Context, userID int64, addSlugs []string, removeSlugs []string, expiresAt *time.Time) error
	GetAllSegmentsData(ctx context.Context, userID int64) ([]model.SegmentUserDataDTO, error)
}

type pgxSegmentRepo struct {
	db *sql.DB
}

func NewPgxSegmentRepo(db *sql.DB) SegmentRepo {
	return &pgxSegmentRepo{db: db}
}

func (r *pgxSegmentRepo) CreateSegment(ctx context.Context, slug string, auto_percent *int) error {
	var err error
	if auto_percent != nil {
		_, err = r.db.ExecContext(ctx, "INSERT INTO segments(slug, auto_percent) VALUES($1, $2)", slug, auto_percent)
	} else {
		_, err = r.db.ExecContext(ctx, "INSERT INTO segments(slug) VALUES($1)", slug)
	}
	return err
}

func (r *pgxSegmentRepo) DeleteSegment(ctx context.Context, slug string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM segments WHERE slug = $1", slug)
	return err
}

func (r *pgxSegmentRepo) GetAllSegments(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT slug FROM segments")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var slugs []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		slugs = append(slugs, slug)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", apperror.ErrDuringRowsIteration, err)
	}
	return slugs, nil
}

func (r *pgxSegmentRepo) SegmentExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM segments WHERE slug = $1)", slug).Scan(&exists)
	return exists, err
}

func (r *pgxSegmentRepo) AddUserToSegment(ctx context.Context, userID int64, slug string) error {
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO user_segments(user_id, segment_slug) VALUES($1, $2);
		`,
		userID, slug); err != nil {
		return fmt.Errorf("%w: %w", apperror.ErrCannotInsertT, err)
	}
	return nil
}

func (r *pgxSegmentRepo) GetAllSegmentsData(ctx context.Context, userID int64) ([]model.SegmentUserDataDTO, error) {
	query := `
        SELECT
            s.slug,                   -- 1. Имя сегмента
            s.auto_percent,           -- 2. Процент
            us.expires_at,            -- 3. Время истечения (NULL, если не назначен вручную)
            CASE WHEN us.user_id IS NOT NULL THEN TRUE ELSE FALSE END AS is_manual -- 4. Назначен ли вручную
        FROM 
            segments s
        LEFT JOIN 
            user_segments us 
            ON s.slug = us.segment_slug AND us.user_id = $1; 
    `
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("db query failed: %w", err)
	}
	defer rows.Close()
	var results []model.SegmentUserDataDTO
	for rows.Next() {
		var dto model.SegmentUserDataDTO
		var expiresAt sql.NullTime // Используем sql.NullTime для expires_at, т.к. может быть NULL
		if err := rows.Scan(
			&dto.Slug,
			&dto.AutoPercent,
			&expiresAt,
			&dto.IsManuallyAssigned,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		if expiresAt.Valid {
			dto.ExpiresAt = &expiresAt.Time
		}
		results = append(results, dto)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}
	return results, nil
}

func (r *pgxSegmentRepo) GetHTable(ctx context.Context) ([]model.HistoryTableDTO, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT * FROM user_segment_history")
	if err != nil {
		return nil, fmt.Errorf("db query failed: %w", err)
	}
	defer rows.Close()
	var historyTables []model.HistoryTableDTO
	for rows.Next() {
		var historyTable model.HistoryTableDTO
		if err := rows.Scan(&historyTable); err != nil {
			return nil, apperror.ErrSegmentNotFound
		}
		historyTables = append(historyTables, historyTable)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.ErrDuringRowsIteration
	}
	return historyTables, nil
}

func (r *pgxSegmentRepo) GetHForPeriod(ctx context.Context, year, month int) ([]model.HistoryTableDTO, error) {
	// Начало месяца (например, 2025-10-01 00:00:00)
	startTime := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	// Начало следующего месяца (например, 2025-11-01 00:00:00)
	endTime := startTime.AddDate(0, 1, 0)
	rows, err := r.db.QueryContext(ctx, `
        SELECT id, user_id, segment_slug, operation, created_at 
        FROM user_segment_history
        WHERE created_at >= $1 AND created_at < $2
        ORDER BY created_at
    `, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("db query failed for report: %w", err)
	}
	defer rows.Close()
	var historyTables []model.HistoryTableDTO
	for rows.Next() {
		var dto model.HistoryTableDTO
		if err := rows.Scan(&dto.ID, &dto.User_ID, &dto.Segment_slug, &dto.Operation, &dto.Created_at); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		historyTables = append(historyTables, dto)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", apperror.ErrDuringRowsIteration, err)
	}
	return historyTables, nil
}

func (r *pgxSegmentRepo) UpdateUserSegments(ctx context.Context, userID int64, addSlugs []string, removeSlugs []string, expiresAt *time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", apperror.ErrFailedBTransaction, err)
	}
	defer tx.Rollback()
	stmtRemove, err := tx.PrepareContext(ctx, "DELETE FROM user_segments WHERE user_id = $1 AND segment_slug = $2")
	if err != nil {
		return fmt.Errorf("failed to prepare remove statement: %w", err)
	}
	defer stmtRemove.Close()
	stmtAdd, err := tx.PrepareContext(ctx, `
        INSERT INTO user_segments (user_id, segment_slug, expires_at) 
        VALUES ($1, $2, $3) 
        ON CONFLICT (user_id, segment_slug) 
        DO UPDATE SET expires_at = EXCLUDED.expires_at; 
    `)
	if err != nil {
		return fmt.Errorf("failed to prepare add statement: %w", err)
	}
	defer stmtAdd.Close()
	for _, slug := range removeSlugs {
		if _, err := stmtRemove.ExecContext(ctx, userID, slug); err != nil {
			return fmt.Errorf("%w: %w", apperror.ErrCannotDeleteFT, err)
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO user_segment_history(user_id, segment_slug, operation) VALUES($1, $2, $3)", userID, slug, "REMOVED"); err != nil {
			return fmt.Errorf("%w: %w", apperror.ErrCannotInsertT, err)
		}
	}
	for _, slug := range addSlugs {
		if _, err := stmtAdd.ExecContext(ctx, userID, slug, expiresAt); err != nil {
			return fmt.Errorf("%w: %w", apperror.ErrCannotInsertT, err)
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO user_segment_history(user_id, segment_slug, operation) VALUES($1, $2, $3)", userID, slug, "ADDED"); err != nil {
			return fmt.Errorf("%w: %w", apperror.ErrCannotInsertT, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: %w", apperror.ErrFailedCTransaction, err)
	}
	return nil
}
