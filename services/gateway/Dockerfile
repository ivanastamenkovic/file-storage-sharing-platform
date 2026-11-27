FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY services ./services
RUN CGO_ENABLED=0 go build -o /out/app ./services/gateway

FROM alpine:3.20
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/app /app
ENTRYPOINT ["/app"]
