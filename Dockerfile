FROM golang:1.23-alpine AS builder 
WORKDIR /app

# Копируем только то, что нужно для сборки
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код
COPY . . 

# Собираем статический исполняемый файл (не зависит от системных библиотек)
RUN go build -ldflags "-s -w" -o /go-service ./cmd/main.go 

# Используем минимальный базовый образ (без компилятора, без ОС!)
FROM scratch 

# scratch — это самый маленький образ (около 4 МБ)
# Если нужны системные библиотеки (как у нас)
# FROM alpine:latest # можно использовать alpine (5 МБ)

# Копируем ТОЛЬКО скомпилированный бинарник из первого этапа
COPY --from=builder /go-service /go-service

ENTRYPOINT ["/go-service"]