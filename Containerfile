# ~/.GH/Qompass/Go/beacon/Containerfile
# -------------------------------------
# Copyright (C) 2025 Qompass AI, All rights reserved

FROM nixos/nix:25.05 AS builder

ARG UID=1000
ARG GID=1000

RUN addgroup -g $GID qai && \
    adduser -D -u $UID -G qai beacon

WORKDIR /build
RUN chown beacon:qai /build
USER beacon

COPY --chown=beacon:qai default.nix .

RUN nix-build -E 'with import <nixpkgs> {}; callPackage ./default.nix {}'

COPY --chown=beacon:qai . .

RUN nix-build

FROM nixos/nix:25.05-minimal

ARG UID=1000
ARG GID=1000

RUN addgroup -g $GID qai && \
    adduser -D -u $UID -G qai beacon

WORKDIR /beacon
RUN chown beacon:qai /beacon

COPY --from=builder --chown=beacon:qai /build/result/bin/beacon /bin/beacon

USER root
RUN nix-env -iA \
    nixpkgs.glibcLocales \
    nixpkgs.tzdata && \
    chmod 755 /bin/beacon

USER beacon

ENV TZDIR="/etc/zoneinfo" \
    XDG_CACHE_HOME="/beacon/.cache" \
    XDG_CONFIG_HOME="/beacon/.config"

EXPOSE 8025/tcp
EXPOSE 8080/tcp
EXPOSE 8443/tcp
EXPOSE 9093/tcp

CMD ["/bin/beacon", "serve"]
