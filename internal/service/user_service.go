package service

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"progression1/internal/apperror"
	"progression1/internal/model"
	"progression1/internal/repository"
	"regexp"
	"time"
)

type UserService struct {
	segRepo repository.SegmentRepo // ← зависимость через интерфейс
}

func NewUserService(segRepo repository.SegmentRepo) *UserService {
	return &UserService{segRepo: segRepo}
}

func percentValidate(auto_percent *int) error {
	if auto_percent != nil {
		if *auto_percent < 0 {
			return apperror.ErrPercentAbove
		} else if *auto_percent > 100 {
			return apperror.ErrPercentLess
		}
	}
	return nil
}
func (s *UserService) CreateSegment(ctx context.Context, slug string, auto_percent *int) error {
	if err := slugValidate(slug); err != nil {
		return err
	}
	if err := percentValidate(auto_percent); err != nil {
		return err
	}
	if err := s.segRepo.CreateSegment(ctx, slug, auto_percent); err != nil {
		return err
	}
	return nil
}

func (s *UserService) GetAllSegments(ctx context.Context) ([]string, error) {
	segments, err := s.segRepo.GetAllSegments(ctx)
	if err != nil {
		return nil, err
	}
	return segments, nil
}

func (s *UserService) SegmentExists(ctx context.Context, slug string) (bool, error) {
	exists, err := s.segRepo.SegmentExists(ctx, slug)
	return exists, err
}

func (s *UserService) GetHTable(ctx context.Context) ([]model.HistoryTableDTO, error) {
	historyTables, err := s.segRepo.GetHTable(ctx)
	return historyTables, err
}

func (s *UserService) GetHForPeriod(ctx context.Context, year, month int) ([]model.HistoryTableDTO, error) {
	if err := yearAmonthValidate(year, month); err != nil {
		return nil, err
	}
	historyTables, err := s.segRepo.GetHForPeriod(ctx, year, month)
	return historyTables, err
}

func (s *UserService) UpdateUserSegments(ctx context.Context, userID int64, addSlugs []string, removeSlugs []string, ttlHours *int) error {
	if len(addSlugs) > 100 || len(removeSlugs) > 100 {
		return apperror.ErrTooManySegments
	}
	var expiresAt *time.Time
	if ttlHours != nil && *ttlHours > 0 && *ttlHours < 720 {
		expirationTime := time.Now().Add(time.Duration(*ttlHours) * time.Hour)
		expiresAt = &expirationTime
	}
	for _, slug := range addSlugs {
		if err := slugValidate(slug); err != nil {
			return fmt.Errorf("%w: %w", apperror.ErrCannotInsertT, err)
		}
	}
	for _, slug := range removeSlugs {
		if err := slugValidate(slug); err != nil {
			return fmt.Errorf("%w: %w", apperror.ErrCannotDeleteFT, err)
		}
	}
	slugsToAdd := make(map[string]struct{}, len(addSlugs))
	for _, slug := range addSlugs {
		slugsToAdd[slug] = struct{}{}
	}
	for _, slug := range removeSlugs {
		if _, ok := slugsToAdd[slug]; ok {
			return fmt.Errorf("%w: %s", apperror.ErrSegmentConflict, slug)
		}
	}
	if err := s.segRepo.UpdateUserSegments(ctx, userID, addSlugs, removeSlugs, expiresAt); err != nil {
		return err
	}
	return nil
}

func (s *UserService) AddUserToSegment(ctx context.Context, userID int64, slug string) error {
	if err := userValidate(userID, slug); err != nil {
		return err
	}
	exists, err := s.segRepo.SegmentExists(ctx, slug)
	if err != nil {
		return err
	}
	if !exists {
		return apperror.ErrSlugNotFound
	}
	if err := s.segRepo.AddUserToSegment(ctx, userID, slug); err != nil {
		return err
	}
	return nil
}

func calculateDeterministicBucket(userID int64, slug string) int {
	seed := fmt.Sprintf("%d:%s", userID, slug)
	h := fnv.New32a()
	h.Write([]byte(seed))
	hashValue := h.Sum32()
	return int(hashValue) % 100
}

func (s *UserService) GetUserSegments(ctx context.Context, userID int64) ([]model.SegmentUserDataDTO, error) {
	if userID <= 0 {
		return nil, apperror.ErrUserIDInvalid
	}
	userSegments, err := s.segRepo.GetAllSegmentsData(ctx, userID)
	if err != nil {
		return nil, err
	}
	var activeDTOs []model.SegmentUserDataDTO
	for _, userSegment := range userSegments {
		isExpired := userSegment.ExpiresAt != nil && userSegment.ExpiresAt.Before(time.Now())
		if userSegment.IsManuallyAssigned {
			if !isExpired {
				activeDTOs = append(activeDTOs, userSegment)
			}
			continue
		}
		if userSegment.AutoPercent > 0 {
			bucket := calculateDeterministicBucket(userID, userSegment.Slug)
			if bucket < userSegment.AutoPercent {
				activeDTOs = append(activeDTOs, userSegment)
			}
		}
	}
	return activeDTOs, nil
}

func yearAmonthValidate(year, month int) error {
	if month < 1 || month > 12 {
		return errors.New("month must be between 1 and 12")
	}
	if year < 2000 {
		return fmt.Errorf("year %d is earlier than the minimum allowed year (%d)", year, 2000)
	}
	targetDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	currentMonthStart := time.Now().Truncate(24*time.Hour).AddDate(0, 0, -(time.Now().Day() - 1))
	if targetDate.After(currentMonthStart) {
		return fmt.Errorf("report date (%04d-%02d) cannot be in the future or current incomplete month", year, month)
	}
	return nil
}

func userValidate(userID int64, slug string) error {
	if userID <= 0 {
		return apperror.ErrUserIDInvalid
	}
	if slug == "" {
		return apperror.ErrEmptySlug
	}
	return nil
}

func slugValidate(slug string) error {
	if slug == "" {
		return apperror.ErrEmptySlug
	}
	if len(slug) < 3 || len(slug) > 50 {
		return apperror.ErrSlugLength
	}
	var slugRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !slugRegex.MatchString(slug) {
		return apperror.ErrSlugRegex
	}
	return nil
}
