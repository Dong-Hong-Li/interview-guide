# syntax=docker/dockerfile:1

ARG GO_VERSION=1.24
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETARCH
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o /bin/server ./cmd/server

FROM alpine:3.21 AS final
RUN apk add --no-cache ca-certificates tzdata poppler-utils \
	&& echo "https://dl-cdn.alpinelinux.org/alpine/v3.21/community" >> /etc/apk/repositories \
	&& apk add --no-cache font-noto-cjk \
	&& adduser -D -H -u 10001 appuser
WORKDIR /app
COPY --from=build /bin/server /bin/server
USER appuser
EXPOSE 8080
ENTRYPOINT ["/bin/server"]
