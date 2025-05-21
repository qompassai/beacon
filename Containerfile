# ~/.GH/Qompass/Go/beacon/Containerfile
# -------------------------------------
# Copyright (C) 2025 Qompass AI, All rights reserved

FROM nixos/nix:25.05 AS builder

ARG UID=1000
ARG GID=1000
RUN addgroup -g $GID qai && \
    adduser -D -u $UID -G qai beacon

WORKDIR /build
COPY --chown=beacon:qai . .

RUN nix-build -A beacon

FROM nixos/nix:25.05-minimal

ARG UID=1000
ARG GID=1000
RUN addgroup -g $GID qai && \
    adduser -D -u $UID -G qai beacon

WORKDIR /beacon
COPY --from=builder --chown=beacon:qai /build/result/bin/beacon /bin/beacon

USER root
RUN nix-env -iA \
    nixpkgs.glibcLocales \
    nixpkgs.tzdata && \
    chmod 755 /bin/beacon

USER beacon
EXPOSE 8025 8080 8443 9093
CMD ["/bin/beacon", "serve"]
