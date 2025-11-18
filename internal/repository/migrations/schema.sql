-- Таблица сегментов
CREATE TABLE IF NOT EXISTS segments (
    id SERIAL PRIMARY KEY,
    slug TEXT UNIQUE NOT NULL,
    auto_percent INTEGER NULL CHECK (auto_percent >= 0 AND auto_percent <= 100)
);

-- Таблица связей пользователей и сегментов
CREATE TABLE IF NOT EXISTS user_segments (
    user_id BIGINT NOT NULL,
    segment_id INTEGER NOT NULL REFERENCES segments(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NULL,
    PRIMARY KEY (user_id, segment_id)
);

-- Таблица для истории операций
CREATE TABLE IF NOT EXISTS operation_history (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    segment_slug TEXT NOT NULL,
    operation_type TEXT NOT NULL, -- 'ADDED' или 'REMOVED'
    operation_time TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);