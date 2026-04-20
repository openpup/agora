FROM node:24-alpine AS web-build
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web ./
RUN npm run build

FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/server ./cmd/server

FROM alpine:3.21
WORKDIR /app
COPY --from=build /bin/server /app/server
COPY configs /app/configs
COPY --from=web-build /app/web/dist /app/web/dist
EXPOSE 8080
CMD ["/app/server"]
