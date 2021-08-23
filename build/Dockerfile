# Start by building the application.
FROM golang:buster as build

WORKDIR /go/src/app
ADD . /go/src/app

RUN go get -d -v ./...

RUN go build -o /go/bin/app

# Now copy it into our base image.
FROM gcr.io/distroless/base-debian10
COPY --from=build /go/bin/app /
CMD ["/app"]
ENV CONNECTION_STRING "postgres://root@localhost:26257/postgres?sslmode=disable"
ENV DOWNLOAD_DIR "/files"
ENV HISTORY_LENGTH 1000
ENV RSS_PERIOD "30s"
ENV DISCORD_TOKEN ""