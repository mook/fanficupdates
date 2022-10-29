# syntax=docker/dockerfile:1
# Calibre base image

FROM registry.opensuse.org/opensuse/tumbleweed as base

VOLUME [ "/library" ]

RUN --mount=type=cache,target=/var/cache/zypp,sharing=locked \
    zypper --non-interactive install \
        calibre \
        catatonit \
        python3-html5lib \
        python3-tld

# Build FanFicUpdates
FROM registry.opensuse.org/opensuse/golang:1.18 AS builder
WORKDIR /go/src/github.com/mook/fanficupdates
COPY --link . .
RUN go build -v github.com/mook/fanficupdates

# Install FanFicFare
FROM base AS installer

RUN --mount=type=cache,target=/var/cache/zypp,sharing=locked \
    zypper --non-interactive install curl jq
RUN curl --silent https://api.github.com/repos/JimmXinu/FanFicFare/releases/latest \
        | jq --raw-output '.assets | map(select(.name | contains("Plugin"))) | .[0].browser_download_url' \
        | xargs curl --location --output FanFicFare.zip
RUN mkdir /settings
RUN /usr/bin/env CALIBRE_CONFIG_DIRECTORY=/settings \
        calibre-customize --add-plugin=FanFicFare.zip

# Result image

FROM base
COPY --link --from=installer /settings/ /settings/
COPY --link --from=builder /go/src/github.com/mook/fanficupdates/fanficupdates /usr/local/bin/fanficupdates
WORKDIR /
ENTRYPOINT [ "/usr/bin/catatonit", "--", "/usr/local/bin/fanficupdates", "--settings=/settings", "--library=/library" ]
