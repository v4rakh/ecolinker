#
# Build image
#
FROM golang:1.25-alpine AS builder-server

# Enable automatic toolchain download for Go 1.25+
ENV GOTOOLCHAIN=auto

WORKDIR /src

# Download dependencies first (better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -tags prod -trimpath -ldflags="-s -w" -o /ecolinker ./cmd/ecolinker

#
# Actual image
#
FROM gcr.io/distroless/static-debian13:nonroot

# Copy binary
COPY --from=builder-server /ecolinker /usr/local/bin/ecolinker

# Labels
LABEL maintainer="Varakh <varakh@varakh.de>" \
    description="ecolinker" \
    org.opencontainers.image.authors="Varakh" \
    org.opencontainers.image.vendor="Varakh" \
    org.opencontainers.image.title="ecolinker" \
    org.opencontainers.image.description="ecolinker" \
    org.opencontainers.image.base.name="gcr.io/distroless/static-debian13:nonroot" \
    org.opencontainers.image.source="https://git.myservermanager.com/varakh/ecolinker"

# Expose HTTP port
ENV SERVER_PORT=8080
EXPOSE ${SERVER_PORT}

# Default command
ENTRYPOINT ["/usr/local/bin/ecolinker"]
CMD ["server", "serve"]
