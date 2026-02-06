FROM golang:1.25.5-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

RUN CGO_ENABLE=0 GOOS=linux go build -v -o data-parser ./cmd/data-parser/main.go

FROM gcr.io/distroless/static-debian13

COPY --from=builder /app/data-parser /

ENTRYPOINT ["./data-parser"]
