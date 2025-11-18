package model

import "time"

type HistoryTableDTO struct {
	ID           int
	User_ID      int
	Segment_slug string
	Operation    string
	Created_at   time.Time
}

type ErrorDTO struct {
	Error string `json:"error"`
}

type SegmentDTO struct {
	Slug         string `json:"slug"`
	Auto_percent *int   `json:"auto_percent,omitempty"`
}

type SegmentUpdateUserDTO struct {
	AddSlugs    []string `json:"addslugs"`
	RemoveSlugs []string `json:"removeslugs"`
	TTLHours    *int     `json:"ttl_hours,omitempty"`
}

type UserResponseDTO struct {
	Received bool
	UserID   int64
	Slug     string
}

type SegmentUserDataDTO struct {
	Slug        string // Имя сегмента (из segments)
	AutoPercent int    // Процент для автоматического назначения (из segments)
	// Эти поля будут NULL/false, если пользователь не состоит в сегменте вручную
	IsManuallyAssigned bool
	ExpiresAt          *time.Time
}
