FROM golang:1.24.0-alpine as builder

WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .
RUN go build main.go

FROM golang:1.24.0-alpine as runner

WORKDIR /app

COPY --from=builder /app/main ./main
COPY db/migration ./db/migration
COPY i18n/locales ./i18n/locales

CMD ["/app/main"]
