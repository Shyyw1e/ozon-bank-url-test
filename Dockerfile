FROM golang:1.24 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /server ./cmd/server

FROM gcr.io/distroless/base-debian12:nonroot
ENV HTTP_ADDR=:8080 LOG_LEVEL=info STORAGE_BACKEND=memory
EXPOSE 8080
COPY --from=build /server /server
USER nonroot
ENTRYPOINT ["/server"]
