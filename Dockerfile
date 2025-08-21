# ╔═════════════════════════════════════════════════════╗
# ║                       BUILD                         ║
# ╚═════════════════════════════════════════════════════╝
FROM 11notes/go:1.24 AS build

RUN apk add --no-cache git upx tzdata

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build stripped binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-s -w -extldflags '-static'" \
    -o /out/ownarr ./main.go

# Compress with UPX
RUN upx -q --no-backup -9 --lzma /out/ownarr

# ╔═════════════════════════════════════════════════════╗
# ║                       IMAGE                         ║
# ╚═════════════════════════════════════════════════════╝
FROM scratch

# Copy binary
COPY --from=build /out/ownarr /ownarr

# Copy timezone database
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo

ENTRYPOINT ["/ownarr"]
