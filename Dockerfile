# ╔═════════════════════════════════════════════════════╗
# ║                       BUILD                         ║
# ╚═════════════════════════════════════════════════════╝
FROM 11notes/go:1.24 AS build

RUN apk add --no-cache upx

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build stripped binary
RUN go build -o /out/ownarr -ldflags "-s -w" ./main.go

# Compress with UPX
RUN upx -q --no-backup -9 --lzma /out/ownarr

# ╔═════════════════════════════════════════════════════╗
# ║                       IMAGE                         ║
# ╚═════════════════════════════════════════════════════╝
FROM scratch

# Copy binary
COPY --from=build /out/ownarr /ownarr

# Copy timezone database for TZ support
COPY --from=build /usr/local/go/lib/time/zoneinfo.zip /zoneinfo.zip
ENV ZONEINFO=/zoneinfo.zip

ENTRYPOINT ["/ownarr"]
