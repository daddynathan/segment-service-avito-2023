package https

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"progression1/internal/model"
	"progression1/internal/service"
	"strconv"
	"strings"
)

type HTTPHandlers struct {
	UserService *service.UserService
}

func NewHTTPHandlers(UserService *service.UserService) *HTTPHandlers {
	return &HTTPHandlers{
		UserService: UserService,
	}
}

// @Summary Получить историю операций с сегментами
// @Description Получает историю добавления/удаления сегментов у пользователя. Если указаны параметры 'year' и 'month', возвращает данные за период. Иначе возвращает всю доступную историю.
// @Tags segment
// @Accept json
// @Produce json
// @Param year query int false "Год для фильтрации истории (например, 2024)"
// @Param month query int false "Месяц для фильтрации истории (1-12)"
// @Success 200 {array} model.HistoryTableDTO "Успешная операция. Возвращает список записей истории."
// @Failure 400 {object} model.ErrorDTO "Невалидный запрос: неверный формат года/месяца или невалидный период."
// @Failure 500 {object} model.ErrorDTO "Внутренняя ошибка сервера"
// @Router /segments/history [get]
func (h *HTTPHandlers) HandleGetH(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query() != nil {
		q := r.URL.Query()
		yearStr := q.Get("year")
		monthStr := q.Get("month")
		year, err := strconv.Atoi(yearStr)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		month, err := strconv.Atoi(monthStr)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		historyTables, err := h.UserService.GetHForPeriod(r.Context(), year, month)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(historyTables); err != nil {
			slog.Warn("failed to encode response", "warn", err)
		}
	} else {
		historyTables, err := h.UserService.GetHTable(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(historyTables); err != nil {
			slog.Warn("failed to encode response", "warn", err)
		}
	}
}

// @Summary Добавить сегмент
// @Description Добавляет сегмент
// @Tags segment
// @Accept json
// @Produce json
// @Param input body model.SegmentDTO true "Параметры для добавления сегмента"
// @Success 200 {string} string "Успешная операция"
// @Failure 500 {object} model.ErrorDTO "Внутренняя ошибка сервера"
// @Failure 400 {object} model.ErrorDTO "Невалидный запрос или конфликт"
// @Router /segments [post]
func (h *HTTPHandlers) HandleAddSegment(w http.ResponseWriter, r *http.Request) {
	var dto model.SegmentDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.UserService.CreateSegment(r.Context(), dto.Slug, dto.Auto_percent); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode("slug successfully added"); err != nil {
		slog.Warn("failed to encode response", "warn", err)
	}
}

// @Summary Получить все сегменты
// @Description Получает все существующие сегменты
// @Tags segment
// @Accept json
// @Produce json
// @Success 200 {array} string "Успешная операция"
// @Failure 500 {object} model.ErrorDTO "Внутренняя ошибка сервера при получении сегментов"
// @Router /segments [get]
func (h *HTTPHandlers) HandleGetAllSegments(w http.ResponseWriter, r *http.Request) {
	slugs, err := h.UserService.GetAllSegments(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to fetch segments")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(slugs); err != nil {
		slog.Warn("failed to encode response", "warn", err)
	}
}

func geIDfFromPath(path string) (int64, error) {
	if !strings.HasPrefix(path, "/users/") {
		return 0, errors.New("invalid path")
	}
	// Извлекаем ID: /users/1000/segments → "1000"
	parts := strings.Split(strings.TrimPrefix(path, "/users/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return 0, errors.New("missing user id")
	}
	userID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

// @Summary Обновить сегменты пользователю
// @Description Добавляет или удаляет сегменты у пользователя, опционально устанавливая TTL.
// @Tags user
// @Accept json
// @Produce json
// @Param input body model.SegmentUpdateUserDTO true "Параметры для добавления/удаления сегментов у пользователя"
// @Param user_id path int true "ID пользователя"
// @Success 200 {string} string "Операция прошла успешно"
// @Failure 400 {object} model.ErrorDTO "Невалидный запрос или конфликт"
// @Router /user/{user_id} [patch]
func (h *HTTPHandlers) HandleUpdateUserSegments(w http.ResponseWriter, r *http.Request) {
	userID, err := geIDfFromPath(r.URL.Path)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	var dto model.SegmentUpdateUserDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.UserService.UpdateUserSegments(r.Context(), userID, dto.AddSlugs, dto.RemoveSlugs, dto.TTLHours); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode("user segments successfully updated"); err != nil {
		slog.Warn("failed to encode response", "warn", err)
	}
}

// @Summary Добавить пользователя к сегменту
// @Description Добавляет пользователя к существующему сегменту
// @Tags user
// @Accept json
// @Produce json
// @Param input body model.SegmentDTO true "Параметры для добавления/удаления сегментов"
// @Param user_id path int true "ID пользователя"
// @Success 200 {object} model.UserResponseDTO "Успешная операция"
// @Failure 400 {object} model.ErrorDTO "Невалидный запрос"
// @Router /user/{user_id} [post]
func (h *HTTPHandlers) HandleAddUserToSegment(w http.ResponseWriter, r *http.Request) {
	userID, err := geIDfFromPath(r.URL.Path)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	var dto model.SegmentDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.UserService.AddUserToSegment(r.Context(), userID, dto.Slug); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var resp model.UserResponseDTO = model.UserResponseDTO{
		Received: true,
		UserID:   userID,
		Slug:     dto.Slug,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Warn("failed to encode response", "warn", err)
	}
}

// @Summary Получить сегменты пользователя
// @Description Получает активные сегменты пользователя
// @Tags user
// @Accept json
// @Produce json
// @Param user_id path int true "ID пользователя"
// @Success 200 {array} model.SegmentUserDataDTO "Список активных сегментов пользователя"
// @Failure 400 {object} model.ErrorDTO "Невалидный ID пользователя"
// @Failure 404 {object} model.ErrorDTO "Пользователь не найден"
// @Failure 500 {object} model.ErrorDTO "Внутренняя ошибка сервера"
// @Router /user/{user_id} [get]
func (h *HTTPHandlers) HandleGetUserSegments(w http.ResponseWriter, r *http.Request) {
	userID, err := geIDfFromPath(r.URL.Path)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	slugs, err := h.UserService.GetUserSegments(r.Context(), userID)
	if err != nil {
		if errors.Is(err, errors.New("user not found")) {
			writeJSONError(w, http.StatusNotFound, err.Error())
		} else if errors.Is(err, errors.New("segment not found")) {
			writeJSONError(w, http.StatusBadRequest, err.Error())
		} else {
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(slugs); err != nil {
		slog.Warn("failed to encode response", "warn", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(model.ErrorDTO{Error: message})
}
