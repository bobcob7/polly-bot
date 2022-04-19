FROM golang:1.18-alpine as builder
# Create appuser.
ENV USER=appuser
ENV UID=10001 
# See https://stackoverflow.com/a/55757473/12429735RUN 
RUN adduser \    
    --disabled-password \    
    --gecos "" \    
    --home "/nonexistent" \    
    --shell "/sbin/nologin" \    
    --no-create-home \    
    --uid "${UID}" \    
    "${USER}"
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates
# RUN apt-get update && apt-get install git ca-certificates && update-ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify
COPY . .
RUN go build -o polly .
# RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o polly .

FROM alpine:latest
USER appuser:appuser
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /app/polly /usr/bin/polly
CMD ["/usr/bin/polly"]