package service

import (
	"context"
	"errors"
	"progression1/internal/apperror"
	"progression1/internal/model"
	"reflect"
	"sort"
	"testing"
	"time"
)

func (m *MockSegmentRepo) CreateSegment(ctx context.Context, slug string, auto_percent *int) error {
	return nil
}
func (m *MockSegmentRepo) DeleteSegment(ctx context.Context, slug string) error { return nil }
func (m *MockSegmentRepo) GetAllSegments(ctx context.Context) ([]string, error) { return nil, nil }
func (m *MockSegmentRepo) GetHTable(ctx context.Context) ([]model.HistoryTableDTO, error) {
	return nil, nil
}
func (m *MockSegmentRepo) GetHForPeriod(ctx context.Context, year, month int) ([]model.HistoryTableDTO, error) {
	return nil, nil
}
func (m *MockSegmentRepo) SegmentExists(ctx context.Context, slug string) (bool, error) {
	return false, nil
}
func (m *MockSegmentRepo) AddUserToSegment(ctx context.Context, userID int64, slug string) error {
	return nil
}
func (m *MockSegmentRepo) GetAllSegmentsData(ctx context.Context, userID int64) ([]model.SegmentUserDataDTO, error) {
	return m.getAllSegmentsData(ctx, userID)
}
func (m *MockSegmentRepo) UpdateUserSegments(ctx context.Context, userID int64, addSlugs []string, removeSlugs []string, expiresAt *time.Time) error {
	return m.updateUserSegments(ctx, userID, addSlugs, removeSlugs, expiresAt)
}

type MockSegmentRepo struct {
	getAllSegmentsData func(ctx context.Context, userID int64) ([]model.SegmentUserDataDTO, error)
	updateUserSegments func(ctx context.Context, userID int64, addSlugs []string, removeSlugs []string, expiresAt *time.Time) error
}

var (
	ctx        = context.Background()
	timeNow    = time.Now()
	timeFuture = timeNow.Add(time.Hour * 24)
	timePast   = timeNow.Add(time.Hour * -24)
)

func TestSlugValidate(t *testing.T) {
	tests := []struct {
		name    string // Имя теста для t.Run
		input   string // Входной slug
		wantErr bool   // Ожидаем ли мы ошибку (true/false)
	}{
		// Успешные сценарии
		{name: "ValidSimple", input: "AVITO_VOICE", wantErr: false},
		{name: "ValidDigits", input: "test1000", wantErr: false},
		{name: "ValidMinLength", input: "abc", wantErr: false},

		// Ошибочные сценарии
		{name: "ValidMaxLength", input: "a-very-long-slug-with-many-words-and-digits-123", wantErr: true},
		{name: "ErrorEmpty", input: "", wantErr: true},
		{name: "ErrorTooShort", input: "ab", wantErr: true},
		{name: "ErrorSpecialChars", input: "bad-slug!", wantErr: true},
		{name: "ErrorSpaces", input: "with spaces", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := slugValidate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("slugValidate(%s) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
func TestPercentValidate(t *testing.T) {
	int1 := 50
	int2 := -1
	int3 := 101
	int1t := &int1
	int2t := &int2
	int3t := &int3
	tests := []struct {
		name    string // Имя теста для t.Run
		input   *int   // Входной slug
		wantErr bool   // Ожидаем ли мы ошибку (true/false)
	}{
		// Успешные сценарии
		{name: "ValidSimple", input: int1t, wantErr: false},
		{name: "ValidSimple", input: nil, wantErr: false},

		// Ошибочные сценарии
		{name: "ErrorAbove", input: int2t, wantErr: true},
		{name: "ErrorLess", input: int3t, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := percentValidate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("percentValidate(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
func TestUserService_GetUserSegments(t *testing.T) {
	testData := []model.SegmentUserDataDTO{
		// 1. Ручной, постоянный сегмент (Активен)
		{
			Slug: "MANUAL_PERMANENT", IsManuallyAssigned: true, ExpiresAt: nil, AutoPercent: 10,
		},
		// 2. Ручной, но ПРОСРОЧЕННЫЙ сегмент (Неактивен - фильтруется)
		{
			Slug: "MANUAL_EXPIRED_TTL", IsManuallyAssigned: true, ExpiresAt: &timePast, AutoPercent: 0,
		},
		// 3. Автоматический сегмент, в который пользователь попадает (Должен быть активен).
		// Bucket: calculateDeterministicBucket(1000, "AUTO_HIT") = 7. AutoPercent: 10. (7 < 10, HIT)
		{
			Slug: "AUTO_HIT", IsManuallyAssigned: true, ExpiresAt: nil, AutoPercent: 10,
		},
		// 4. Автоматический сегмент, в который пользователь не попадает.
		// Bucket: calculateDeterministicBucket(1000, "AUTO_MISS") = 17. AutoPercent: 10. (17 >= 10, MISS)
		{
			Slug: "AUTO_MISS", IsManuallyAssigned: false, ExpiresAt: nil, AutoPercent: 10,
		},
		// 5. Ручной, но АКТИВНЫЙ TTL сегмент (Должен быть активен)
		{
			Slug: "MANUAL_ACTIVE_TTL", IsManuallyAssigned: true, ExpiresAt: &timeFuture, AutoPercent: 10,
		},
	}
	// Активны только: 1 (Permanent), 3 (Auto Hit), 5 (Active TTL)
	expectedSlugs := []string{"MANUAL_PERMANENT", "AUTO_HIT", "MANUAL_ACTIVE_TTL"}
	mockRepo := &MockSegmentRepo{
		getAllSegmentsData: func(ctx context.Context, userID int64) ([]model.SegmentUserDataDTO, error) {
			if userID == 1000 {
				return testData, nil
			}
			return nil, errors.New("user not found from mock db")
		},
	}
	userService := NewUserService(mockRepo)
	t.Run("Success_FilteringTTLAndHashing", func(t *testing.T) {
		activeSegments, err := userService.GetUserSegments(ctx, 1000)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		var actualSlugs []string
		for _, activeSegment := range activeSegments {
			actualSlugs = append(actualSlugs, activeSegment.Slug)
		}
		// Сортируем оба списка для надежного сравнения, так как порядок может быть не гарантирован.
		sort.Strings(actualSlugs)
		sort.Strings(expectedSlugs)
		if !reflect.DeepEqual(actualSlugs, expectedSlugs) {
			t.Errorf("Incorrect segments returned.\nExpected: %v\nGot: %v", expectedSlugs, actualSlugs)
		}
	})
	t.Run("Error_InvalidUserID", func(t *testing.T) {
		_, err := userService.GetUserSegments(ctx, 0)
		if err == nil || !errors.Is(err, apperror.ErrUserIDInvalid) {
			t.Errorf("Expected ErrUserIDInvalid, got: %v", err)
		}
	})
	t.Run("HashDeterminismCheck", func(t *testing.T) {
		slug := "TEST_HASH_DETERMINISM"
		userID := int64(9001)
		firstBucket := calculateDeterministicBucket(userID, slug)
		secondBucket := calculateDeterministicBucket(userID, slug)
		if firstBucket != secondBucket {
			t.Errorf("Hash function must be deterministic. Got %d and %d", firstBucket, secondBucket)
		}
	})
}
func TestUserService_UpdateUserSegments(t *testing.T) {
	mockRepo := &MockSegmentRepo{
		updateUserSegments: func(ctx context.Context, userID int64, addSlugs []string, removeSlugs []string, expiresAt *time.Time) error {
			if userID == 1000 {
				return nil
			}
			return errors.New("user not found from mock db")
		},
	}
	userService := NewUserService(mockRepo)
	t.Run("Error_SlugConflict", func(t *testing.T) {
		ttlHours := 24
		ttlHoursF := &ttlHours
		err := userService.UpdateUserSegments(
			ctx,
			1000,
			[]string{"MANUAL_PERMANENT"},
			[]string{"MANUAL_PERMANENT"},
			ttlHoursF,
		)
		if err == nil || !errors.Is(err, apperror.ErrSegmentConflict) {
			t.Fatalf("Expected ErrSegmentConflict, got: %v", err)
		}
	})
	t.Run("Error_SlugValidate", func(t *testing.T) {
		ttlHours := 24
		ttlHoursF := &ttlHours
		err := userService.UpdateUserSegments(
			ctx,
			1000,
			[]string{"averylongslugwithmanywordsanddigits123123123123123123123123123123123123123123123123"},
			[]string{"VOICE_MESSAGE"},
			ttlHoursF,
		)
		if err == nil || !errors.Is(err, apperror.ErrSlugLength) {
			t.Fatalf("Expected ErrSlugLength, got: %v", err)
		}
	})
	t.Run("Error_BulkLimit", func(t *testing.T) {
		ttlHours := 24
		ttlHoursF := &ttlHours
		var addSlugs []string
		for i := 0; i < 101; i++ {
			addSlugs = append(addSlugs, "1234")
		}
		err := userService.UpdateUserSegments(
			ctx,
			1000,
			addSlugs,
			[]string{"VOICE_MESSAGE"},
			ttlHoursF,
		)
		if err == nil || !errors.Is(err, apperror.ErrTooManySegments) {
			t.Fatalf("Expected ErrTooManySegments, got: %v", err)
		}
	})
}
