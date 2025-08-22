# Build stage
FROM golang:1.24-alpine AS build

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-s -w -extldflags '-static'" \
    -o /out/ownarr ./cmd/ownarr

# Compress binary (optional)
RUN apk add --no-cache upx && \
    upx -q --no-backup -9 --lzma /out/ownarr || true

# Final stage
FROM scratch

ENV PORT=8080

# Copy binary
COPY --from=build /out/ownarr /ownarr

# Copy timezone database
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo

# Copy curl for healthcheck
COPY --from=build /usr/bin/curl /usr/bin/curl

HEALTHCHECK --interval=5s --timeout=2s --start-interval=5s \
    CMD ["/usr/bin/curl", "-X", "GET", "-kILs", "--fail", "http://localhost:8080/health"]

ENTRYPOINT ["/ownarr"]
