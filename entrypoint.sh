#!/usr/bin/env bash

persist() {
    local path="${1%/}"
    local src="/state/global/home/${USER}/${path}"
    local dst="${HOME}/${path}"

    if [[ "$1" == */ ]]; then
        mkdir -p "$src" "$(dirname "$dst")"
        ln -sfn "$src" "$dst"
    else
        mkdir -p "$(dirname "$src")" "$(dirname "$dst")"
        touch "$src"
        ln -sf "$src" "$dst"
    fi
}

persist ".local/share/fish/fish_history"

persist ".cache/opencode/"
persist ".local/share/opencode/"
persist ".local/state/opencode/"

persist ".cache/uv/"
persist ".local/share/uv/"

exec "$@"
