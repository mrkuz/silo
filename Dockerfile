FROM alpine

ARG USER
ARG HOME=/home/$USER

RUN apk add --no-cache \
  bash coreutils curl shadow-uidmap sudo xz && \
  rm -rf /var/cache/apk/*

# User setup

RUN adduser -D $USER && \
  adduser $USER wheel && \
  echo '%wheel ALL=(ALL) NOPASSWD: ALL' > /etc/sudoers.d/wheel && \
  chmod 0440 /etc/sudoers.d/wheel

# Rootless containers

RUN echo "$USER:100000:65536" >> /etc/subuid && \
    echo "$USER:100000:65536" >> /etc/subgid

ENV _CONTAINERS_USERNS_CONFIGURED=""

# Nix build users (see: https://hub.docker.com/r/nixos/nix/dockerfile/)

RUN addgroup -g 30000 -S nixbld && \
  for i in $(seq 1 30); do \
    adduser -S -D -h /var/empty \
      -g "Nix build user $i" \
      -u $((30000 + i)) \
      -G nixbld nixbld$i ; \
  done

# Nix configuration

RUN mkdir -p /etc/nix && \
  echo 'sandbox = false' > /etc/nix/nix.conf && \
  echo 'experimental-features = nix-command flakes' >> /etc/nix/nix.conf && \
  echo 'auto-optimise-store = true' >> /etc/nix/nix.conf

# User environment

USER $USER
WORKDIR $HOME

# Install Nix

RUN bash <(curl -L https://nixos.org/nix/install) --no-daemon

ENV PATH=$HOME/.nix-profile/bin:$HOME/.nix-profile/sbin:$PATH
ENV USER=$USER

# Activate home-manager configuration

ADD --chown=$USER:$USER flake.nix $HOME/.config/home-manager/flake.nix
ADD --chown=$USER:$USER home.nix $HOME/.config/home-manager/home.nix

# Install home-manager

RUN nix flake update --override-input nixpkgs nixpkgs --flake $HOME/.config/home-manager && \
    nix run nixpkgs#home-manager -- switch --flake $HOME/.config/home-manager && \
    nix-store --gc

# Entrypoint

USER root
ADD entrypoint.sh /usr/local/bin/entrypoint
RUN chmod +x /usr/local/bin/entrypoint
ENTRYPOINT ["/usr/local/bin/entrypoint"]
CMD ["sleep", "infinity"]
