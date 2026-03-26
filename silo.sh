#!/usr/bin/env bash

username=$(id -u -n)
current_dir=$(basename "$PWD")

silo_toml=".silo.toml"
if [ -f "$silo_toml" ]; then
  silo_id="$(grep -E '^id[[:space:]]*=' "$silo_toml" | sed 's/.*=[[:space:]]*"\(.*\)"/\1/')"
else
  silo_id="$(LC_ALL=C tr -dc 'a-z0-9' < /dev/urandom | head -c 8)"
fi
container_name="silo-$silo_id"

default_cmd="fish --login"

podman_enabled=0
workspace_enabled=1
state_enabled=1

while getopts "pWS" opt; do
  case $opt in
    p) podman_enabled=1 ;;
    W) workspace_enabled=0 ;;
    S) state_enabled=0 ;;
    *) echo "Usage: $0 [-p] [-W] [-S] [--] [podman args...]" >&2; exit 1 ;;
  esac
done
shift $((OPTIND - 1))

extra_args=("$@")

if [ $podman_enabled -eq 1 ]; then
  security_args=(--security-opt label=disable --device /dev/fuse)
else
  security_args=(--cap-drop=ALL --cap-add=NET_BIND_SERVICE --security-opt no-new-privileges)
fi

workspace_args=()
if [ $workspace_enabled -eq 1 ]; then
  workspace_args=(
    --volume "$PWD:/workspace/${silo_id}/${current_dir}:Z"
    --workdir "/workspace/${silo_id}/${current_dir}"
  )
fi

state_args=()
if [ $state_enabled -eq 1 ]; then
  state_args=(--volume "silo-state:/state/global:Z")
fi

toml_array() {
  if [ $# -eq 0 ]; then
    printf '[]'
    return
  fi
  local out="[\n"
  while [ $# -gt 0 ]; do
    local entry="$1"; shift
    if [ $# -gt 0 ] && [[ "$1" != -* ]]; then
      entry="$entry $1"; shift
    fi
    out+="  \"${entry//\"/\\\"}\",\n"
  done
  out+="]"
  printf '%b' "$out"
}

write_toml_sections() {
  local toml_file="$1"
  local bool_workspace bool_state bool_podman
  [ $workspace_enabled -eq 1 ] && bool_workspace="true" || bool_workspace="false"
  [ $state_enabled -eq 1 ]     && bool_state="true"     || bool_state="false"
  [ $podman_enabled -eq 1 ]    && bool_podman="true"     || bool_podman="false"

  local resolved_args=("${security_args[@]}" "${workspace_args[@]}" "${state_args[@]}")
  local args_val extra_val
  args_val=$(toml_array "${resolved_args[@]}")
  extra_val=$(toml_array "${extra_args[@]}")

  printf '[general]\nid = "%s"\nuser = "%s"\ncontainer_name = "%s"\n' \
    "$silo_id" "$username" "$container_name" > "$toml_file"
  printf '\n[features]\nworkspace = %s\nstate = %s\npodman = %s\n' \
    "$bool_workspace" "$bool_state" "$bool_podman" >> "$toml_file"
  printf '\n[podman]\nargs = %s\nextra_args = %s\ncommand = "%s"\n' \
    "$args_val" "$extra_val" "$default_cmd" >> "$toml_file"
}

if [ "${1:-}" = "rm" ]; then
  if podman container exists "$container_name"; then
    echo "Removing $container_name..."
    podman rm -f "$container_name"
  else
    echo "No container $container_name found."
  fi
  exit 0
fi

if [ "${1:-}" = "devcontainer" ]; then
  devcontainer_file=".devcontainer.json"

  json_args=$(printf '"%s", ' "${security_args[@]}")
  run_args="[${json_args%, }]"

  new_content=$(cat <<EOF
{
  "image": "silo",
  "remoteUser": "${username}",
  "runArgs": ${run_args},
  "overrideCommand": false,
  "customizations": {
    "vscode": {
      "settings": {
        "terminal.integrated.defaultProfile.linux": "fish",
        "terminal.integrated.profiles.linux": {
          "fish": { "path": "/home/${username}/.nix-profile/bin/fish", "args": ["--login"] }
        }
      }
    }
  }
}
EOF
)

  if [ -f "$devcontainer_file" ]; then
    diff_output=$(diff -uNr "$devcontainer_file" - <<< "$new_content")
    if [ -z "$diff_output" ]; then
      exit 0
    fi
    echo "$diff_output"
    printf "Replace %s? [y/N] " "$devcontainer_file"
    read -r answer
    case "$answer" in
      [yY]*) ;;
      *) echo "Aborted"; exit 0 ;;
    esac
  fi

  printf '%s\n' "$new_content" > "$devcontainer_file"
  echo "Generated ${devcontainer_file}"
  exit 0
fi

if podman container inspect --format '{{.State.Running}}' "$container_name" 2>/dev/null | grep -q true; then
  echo "Joining $container_name..."
  podman exec -ti "$container_name" $default_cmd
elif podman container exists "$container_name"; then
  echo "Starting $container_name..."
  podman start -ai "$container_name"
else
  echo "Creating and starting $container_name..."
  write_toml_sections "$silo_toml"
  podman run -ti \
    "${security_args[@]}" \
    --name "$container_name" \
    --hostname "$container_name" \
    --user $username \
    "${workspace_args[@]}" \
    "${state_args[@]}" \
    "${extra_args[@]}" \
    "silo" \
    $default_cmd
fi
