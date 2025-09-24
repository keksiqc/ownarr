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

# Install timezone database
RUN apk add --no-cache tzdata

# Final stage
FROM scratch

# Copy binary
COPY --from=build /out/ownarr /ownarr

# Copy timezone database
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo

ENTRYPOINT ["/ownarr"]
