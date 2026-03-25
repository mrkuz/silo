#!/usr/bin/env bash

username=$(id -u -n)
podman build -t silo --build-arg USER=$username .