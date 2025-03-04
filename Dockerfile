# Gunakan argumen untuk versi Go yang fleksibel
ARG GO_VERSION=1.23

# Stage Build: Kompilasi Aplikasi
# Menggunakan alpine variant dari image Go untuk ukuran yang minimal
FROM golang:${GO_VERSION}-alpine AS build

# Set working directory di dalam container
WORKDIR /src

# Salin file dependensi terlebih dahulu untuk optimasi caching
# Dengan cara ini, jika hanya source code yang berubah, 
# dependensi tidak perlu di-download ulang
COPY go.mod go.sum ./

# Download semua dependensi projekt
# Perintah ini akan membaca go.mod dan go.sum untuk resolusi dependensi
RUN go mod download

# Salin seluruh source code projekt
COPY . .

# Build aplikasi dengan konfigurasi optimasi
# - Gunakan cache untuk mempercepat proses build
# - CGO_ENABLED=0 untuk static binary
# - GOOS=linux dan GOARCH=amd64 untuk kompatibilitas
# - ldflags untuk mengurangi ukuran binary dan menghilangkan debug info
RUN --mount=type=cache,target=/go/pkg/mod/ \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/server .

# Stage Akhir: Container Minimal untuk Eksekusi
# Gunakan Alpine Linux sebagai base image produksi
FROM alpine:3.21 AS final

# Instal dependensi runtime yang diperlukan
# - ca-certificates: Untuk koneksi HTTPS yang aman
# - tzdata: Untuk manajemen zona waktu
# Gunakan mount cache untuk mempercepat instalasi paket
RUN --mount=type=cache,target=/var/cache/apk \
    apk --update add \
        ca-certificates \
        tzdata \
        && \
        update-ca-certificates

# Buat user non-privileged untuk menjalankan aplikasi
# Praktik keamanan: Hindari menjalankan aplikasi sebagai root
ARG UID=10001
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    appuser

# Beralih ke user non-privileged
USER appuser

# Salin artifact dari stage build
# - Binary utama aplikasi
# - Folder web/static jika diperlukan
COPY --from=build /bin/server /bin
COPY --from=build /src/web /web

# Salin file konfigurasi jika diperlukan for example ssl
# COPY --from=build /src/server.key /server.key
# COPY --from=build /src/server.crt /server.crt

# If you need healthcheck
# Periksa status aplikasi setiap 30 detik
# Timeout 10 detik untuk response
# 3 percobaan gagal akan dianggap tidak sehat
# Asumsikan aplikasi memiliki endpoint /health
# HEALTHCHECK --interval=30s \
#             --timeout=10s \
#             --start-period=5s \
#             --retries=3 \
#             CMD wget -q --spider http://localhost:3000 || exit 1

# Ekspos port yang digunakan aplikasi
EXPOSE 3003

# Definisikan entrypoint untuk menjalankan server
ENTRYPOINT ["/bin/server"]