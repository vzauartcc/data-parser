FROM golang:1.26.5-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux GOAMD64=v3 go build -v -ldflags="-s -w" -o data-parser ./cmd/data-parser/main.go

FROM gcr.io/distroless/static-debian13

COPY --from=builder /app/data-parser /

ENTRYPOINT ["./data-parser"]
