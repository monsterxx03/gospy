# syntax=docker/dockerfile:1

ARG GO_VERSION=1.21
ARG ALPINE_VERSION=3.18

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder
ARG TARGETARCH
WORKDIR /app
# 复制所有必要文件（新增go.mod）
COPY main.go go.mod ./
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -o server .

FROM alpine:${ALPINE_VERSION}
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
