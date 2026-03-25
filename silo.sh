#!/usr/bin/env bash

username=$(id -u -n)
current_dir=$(basename "$PWD")

silo_toml=".silo.toml"
if [ -f "$silo_toml" ]; then
  silo_id="$(grep -E '^id[[:space:]]*=' "$silo_toml" | sed 's/.*=[[:space:]]*"\(.*\)"/\1/')"
else
  silo_id="$(LC_ALL=C tr -dc 'a-z0-9' < /dev/urandom | head -c 8)"
  printf '[general]\nid = "%s"\n' "$silo_id" > "$silo_toml"
fi
container_name="silo-$silo_id"

podman_enabled=0
workspace_enabled=1
state_enabled=1

while getopts "pWS" opt; do
  case $opt in
    p) podman_enabled=1
       ;;
    W) workspace_enabled=0
       ;;
    S) state_enabled=0
       ;;
    *) echo "Usage: $0 [-p] [-W] [-S]" >&2
       exit 1
       ;;
  esac
done

workspace_args=()
if [ $workspace_enabled -eq 1 ]; then
  workspace_args=(
    --volume "$PWD:/workspace/${silo_id}/${current_dir}:Z"
    --workdir "/workspace/${silo_id}/${current_dir}"
  )
fi

state_args=()
if [ $state_enabled -eq 1 ]; then
  state_args=(
    --volume "silo-state:/state/global:Z"
  )
fi

if [ "${1:-}" = "rm" ]; then
  if podman container exists "$container_name"; then
    echo "Removing $container_name..."
    podman rm -f "$container_name"
  else
    echo "No container $container_name found."
  fi
  exit 0
fi

if podman container inspect --format '{{.State.Running}}' "$container_name" 2>/dev/null | grep -q true; then
  echo "Joining $container_name..."
  podman exec -ti "$container_name" fish --login
elif podman container exists "$container_name"; then
  echo "Starting $container_name..."
  podman start -ai "$container_name"
else
  echo "Creating and starting $container_name..."
  if [ $podman_enabled -eq 1 ]; then
    # Allow nested containers
    podman run -ti \
      --security-opt label=disable \
      --device /dev/fuse \
      --name "$container_name" \
      --hostname "$container_name" \
      --user $username \
      "${workspace_args[@]}" \
      "${state_args[@]}" \
      "silo" \
      fish --login
  else
    podman run -ti \
      --cap-drop=ALL \
      --cap-add=NET_BIND_SERVICE \
      --security-opt no-new-privileges \
      --name "$container_name" \
      --hostname "$container_name" \
      --user $username \
      "${workspace_args[@]}" \
      "${state_args[@]}" \
      "silo" \
      fish --login
  fi
fi
