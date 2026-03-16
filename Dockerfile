#
# Build image
#
FROM alpine:3.23 AS builder
LABEL maintainer="Varakh <varakh@varakh.de>"

RUN apk --update upgrade && \
    apk add git && \
    apk add go gcc make && \
    rm -rf /var/cache/apk/*

WORKDIR /app
COPY . .
RUN CC=gcc make clean dependencies build-linux-amd64

#
# Actual image
#
FROM alpine:3.23
LABEL maintainer="Varakh <varakh@varakh.de>" \
    description="ecolinker" \
    org.opencontainers.image.authors="Varakh" \
    org.opencontainers.image.vendor="Varakh" \
    org.opencontainers.image.title="ecolinker" \
    org.opencontainers.image.description="ecolinker" \
    org.opencontainers.image.base.name="alpine:3.23"

ENV USER=appuser
ENV GROUP=appuser
ENV UID=2033
ENV GID=2033

RUN apk --update upgrade && \
    apk add tzdata && \
    rm -rf /var/cache/apk/* && \
    addgroup -S ${GROUP} -g ${GID} && \
    adduser -S ${USER} -G ${GROUP} -u ${UID}

COPY --from=builder /app/bin/ecolinker-linux-amd64 /usr/bin/ecolinker

USER ${USER}

ENV SERVER_PORT=8080
EXPOSE ${SERVER_PORT}
CMD ["/usr/bin/ecolinker", "server", "serve"]
