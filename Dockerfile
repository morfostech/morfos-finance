FROM node:22-alpine AS frontend

WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS backend

WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/morfos-finance ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

FROM alpine:3.22

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend /out/morfos-finance /app/morfos-finance
COPY --from=backend /out/seed /app/seed
COPY --from=frontend /src/frontend/dist /app/frontend

ENV FRONTEND_DIR=/app/frontend
ENV UPLOAD_DIR=/tmp/morfos-finance-uploads
EXPOSE 8080

CMD ["/app/morfos-finance"]
