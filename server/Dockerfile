# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS development
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download
COPY . .
EXPOSE 8080
CMD ["go", "run", "."]

FROM development AS build
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api .

FROM gcr.io/distroless/static-debian12:nonroot AS production
COPY --from=build /out/api /api
ENV HTTP_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/api"]
