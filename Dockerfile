FROM golang:alpine AS builder

WORKDIR /src
COPY . /src

RUN apk add --update --no-cache --no-progress make git \
    && make server

FROM alpine:latest
LABEL org.opencontainers.image.licenses=Apache-2.0
LABEL org.opencontainers.image.source="https://github.com/InscuraApp/InscuraScraper"

COPY --from=builder /src/build/inscurascraper-server .

RUN apk add --update --no-cache --no-progress ca-certificates tzdata

ENV GIN_MODE=release
ENV PORT=8080
ENV TOKEN=""
ENV DSN=""
ENV REQUEST_TIMEOUT=""
ENV DB_MAX_IDLE_CONNS=0
ENV DB_MAX_OPEN_CONNS=0
ENV DB_PREPARED_STMT=0
ENV DB_AUTO_MIGRATE=0
ENV IS_PROVIDER_TMDB__API_TOKEN=""
ENV IS_PROVIDER_FANARTTV__API_KEY=""
ENV IS_PROVIDER_TVDB__API_KEY=""
ENV IS_PROVIDER_TVMAZE__API_KEY=""

EXPOSE 8080

ENTRYPOINT ["/inscurascraper-server"]
