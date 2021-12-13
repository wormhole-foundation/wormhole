# syntax=docker.io/docker/dockerfile:1.3@sha256:42399d4635eddd7a9b8a24be879d2f9a930d0ed040a61324cfdf59ef1357b3b2
FROM registry.fedoraproject.org/fedora:35@sha256:2d697a06d17691e87212cf248f499dd47db2e275dfe642ffca5975353ea89887

RUN dnf -y install sway wayvnc procps chromium novnc hostname patch
RUN dnf -y install https://dl.google.com/linux/direct/google-chrome-stable_current_x86_64.rpm

COPY managed.json /etc/opt/chrome/policies/managed/managed.json

COPY sway.conf /etc/sway/config.d/20-docker.conf

ENV WLR_BACKENDS=headless
ENV WLR_LIBINPUT_NO_DEVICES=1
ENV WAYLAND_DISPLAY=wayland-1
ENV XDG_RUNTIME_DIR=/home/headless/.run
ENV SWAYSOCK=/tmp/sway.sock

RUN useradd -m -s /bin/bash headless

# Python 3.10 compatibility fix for websockify (novnc dependency)
# (Fedora 35 packaging bug that'll be resolved sooner or later)
RUN sed -i 's/fromstring/frombytes/' /usr/lib/python3.10/site-packages/websockify/*.py && \
	sed -i 's/tostring/tobytes/' /usr/lib/python3.10/site-packages/websockify/*.py

USER headless
WORKDIR /home/headless

RUN mkdir -p ~/.config ~/.run
COPY --chown=headless:headless run.sh /home/headless/run.sh
