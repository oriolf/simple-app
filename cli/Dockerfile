FROM golang:1.27 AS builder

RUN mkdir /app
WORKDIR /app
ADD . /app

RUN CGO_ENABLED=1 GOOS=linux go build -a -trimpath -ldflags='-extldflags=-static -w -s' -tags osusergo,netgo,sqlite_omit_load_extension -o main

FROM alpine AS production

WORKDIR /
COPY --from=builder /app/main /main
COPY .env /.env
EXPOSE 8000

CMD ["./main", "http"]


