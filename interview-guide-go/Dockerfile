# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26.2
ARG ALPINE_TAG=edge
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETARCH
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o /bin/server ./cmd/server

# Rolling：含较新的 busybox / nss / freetype / cairo 等安全修订（Scout 在 3.23 上对若干 CVE 仍为 not fixed）。
# 若要可复现发布，改为 alpine:3.23@digest 并可定期重建；或在此处钉死 edge 某日 digest。
FROM alpine:${ALPINE_TAG} AS final
RUN apk update && apk upgrade --no-cache \
	&& apk add --no-cache ca-certificates tzdata poppler-utils font-noto-cjk \
	&& adduser -D -H -u 10001 appuser
WORKDIR /app
COPY --from=build /bin/server /bin/server
# 配置不写进镜像：`docker compose` 用 compose.yaml env_file；单独 run 时请挂载宿主 .env：
#   -e ENV_FILE=/config/.env -v /绝对路径/.env:/config/.env:ro
USER appuser
EXPOSE 8080
ENTRYPOINT ["/bin/server"]
