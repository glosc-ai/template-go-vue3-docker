# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS development
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ .
EXPOSE 8080
CMD ["go", "run", "."]

FROM node:22-alpine AS web-build
WORKDIR /web
ARG VITE_API_BASE_URL=/api/v1
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM development AS build
COPY --from=web-build /web/dist ./webassets/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api .

FROM gcr.io/distroless/static-debian12:nonroot AS production
COPY --from=build /out/api /api
ENV HTTP_ADDR=:8080
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/api", "-healthcheck"]
ENTRYPOINT ["/api"]
