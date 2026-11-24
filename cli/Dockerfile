FROM golang:1.27 AS builder

RUN mkdir /app
WORKDIR /app
ADD . /app

RUN CGO_ENABLED=1 GOOS=linux go build -a -trimpath -ldflags='-extldflags=-static -w -s' -tags osusergo,netgo,sqlite_omit_load_extension -o main
EXPOSE 8000

CMD ["./main"]

FROM alpine AS production

RUN mkdir data
WORKDIR /data
COPY --from=builder /app/main /data
EXPOSE 8000

CMD ["./data/main"]


