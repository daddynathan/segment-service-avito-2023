package repository_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"progression1/internal/repository"
	"strconv"
	"sync"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var testDB *sql.DB

const (
	NumGoroutines = 100 // Количество одновременно запускаемых горутин
)

func TestMain(m *testing.M) {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("Переменная DATABASE_URL не установлена. Запустите docker-compose up.")
	}
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("Не удалось создать подключение к тестовой БД: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("Не удалось подключиться к тестовой БД: %v", err)
	}
	testDB = db
	exitCode := m.Run()
	testDB.Close()
	os.Exit(exitCode)
}

func setupTest(t *testing.T) (*sql.DB, repository.SegmentRepo) {
	if testDB == nil {
		t.Skip("Skipping test because global testDB connection is not initialized (TestMain failed or DATABASE_URL is missing).")
		return nil, nil
	}
	db := testDB
	_, err := db.ExecContext(context.Background(), "DELETE FROM segments")
	if err != nil {
		t.Fatalf("Не удалось очистить таблицу segments: %v", err)
	}
	return db, repository.NewPgxSegmentRepo(db)
}

func TestConcurrentSegmentCreation(t *testing.T) {
	db, repo := setupTest(t)
	var wg sync.WaitGroup
	ctx := context.Background()
	// 1. Запуск 100 горутин
	for i := 0; i < NumGoroutines; i++ {
		wg.Add(1)
		// Создаем уникальный SLUG для каждого сегмента
		slug := "CONCURRENT_SEGMENT_" + strconv.Itoa(i)
		go func(slug string) {
			defer wg.Done()
			// 2. Вызов функции создания сегмента
			// NOTE: Нужно передать nil в качестве auto_percent, как в вашей функции CreateSegment
			err := repo.CreateSegment(ctx, slug, nil)
			if err != nil {
				t.Errorf("Goroutine failed to create segment %s: %v", slug, err)
			}
		}(slug)
	}
	wg.Wait()
	// 3. Проверка результата: Должно быть ровно 100 сегментов
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM segments").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count segments after concurrency test: %v", err)
	}
	if count != NumGoroutines {
		t.Errorf("Concurrency test failed: Expected %d segments, got %d. Check DB locks or unique constraint violations.", NumGoroutines, count)
	} else {
		log.Printf("SUCCESS: Concurrency test passed. Created %d unique segments.", count)
	}
}

func TestRepository_UpdateUserSegments_Success(t *testing.T) {
	ctx := context.Background()
	slugToAdd := "AVITO_TEST_1"
	userID := int64(9999)
	_, err := testDB.ExecContext(ctx, "INSERT INTO segments (slug) VALUES ($1)", slugToAdd)
	if err != nil {
		t.Fatalf("Не удалось создать тестовый сегмент (ошибка: %v)", err)
	}
	defer testDB.ExecContext(ctx, "DELETE FROM segments WHERE slug = $1", slugToAdd)
	repo := repository.NewPgxSegmentRepo(testDB)
	//duration := time.Duration(1) * time.Hour
	//expiresAt := time.Now().Add(duration)
	err = repo.UpdateUserSegments(ctx, userID, []string{slugToAdd}, []string{}, nil)
	if err != nil {
		t.Fatalf("UpdateUserSegments упал с ошибкой: %v", err)
	}
	var count int
	err = testDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_segments WHERE user_id = $1 AND segment_slug = $2", userID, slugToAdd).Scan(&count)
	if err != nil {
		t.Fatalf("Не удалось проверить сегмент: %v", err)
	}
	if count != 1 {
		t.Errorf("Ожидалось 1 добавленный сегмент, получено %d", count)
	}
	// НЕ ВЫЗЫВАЕМ tx.Commit()
	// При выходе defer tx.Rollback() откатит все изменения. База данных останется чистой
}
