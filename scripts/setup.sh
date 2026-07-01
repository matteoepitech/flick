#!/usr/bin/env bash

#
# Interactive first-run setup for Flick.
#

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly ROOT
readonly ENV_FILE="$ROOT/.env"
readonly EXAMPLE_FILE="$ROOT/.env.example"

# --- tiny output helpers ----------------------------------------------------

bold() { printf '\033[1m%s\033[0m\n' "$1"; }
dim()  { printf '\033[2m%s\033[0m\n' "$1"; }
info() { printf '  %s\n' "$1"; }

die() {
  printf '\033[31merror:\033[0m %s\n' "$1" >&2
  exit 1
}

# --- helpers ----------------------------------------------------------------

set_env() {
  local key="$1" value="$2"
  local line out="" found=0

  while IFS= read -r line || [ -n "$line" ]; do
    if [[ "$line" == "$key="* ]]; then
      out+="$key=$value"$'\n'
      found=1
    else
      out+="$line"$'\n'
    fi
  done <<< "$ENV_CONTENT"

  if [ "$found" -eq 0 ]; then
    out+="$key=$value"$'\n'
  fi

  ENV_CONTENT="$out"
}

generate_password() {
  local pw=""

  if command -v openssl >/dev/null 2>&1; then
    pw="$(openssl rand -base64 48 | LC_ALL=C tr -dc 'A-Za-z0-9' | cut -c1-32)"
  elif [ -r /dev/urandom ]; then
    pw="$(head -c 512 /dev/urandom | LC_ALL=C tr -dc 'A-Za-z0-9' | cut -c1-32)"
  else
    die "no secure random source available (need openssl or /dev/urandom)"
  fi

  [ "${#pw}" -eq 32 ] || die "failed to generate a 32-character password"
  printf '%s' "$pw"
}

# --- interactive steps ------------------------------------------------------

ask_site_address() {
  local answer domain
  dim "How do you want to expose Flick to the outside?"
  read -rp "Do you already run a reverse proxy in front of Flick (Nginx Proxy Manager, Traefik, Caddy...) ? [y/N] " answer

  if [[ "$answer" =~ ^[Yy]$ ]]; then
    SITE_ADDRESS=":80"
    SITE_SUMMARY="serve plain HTTP on port 80 (your reverse proxy handles TLS)"
    return
  fi

  while true; do
    read -rp "Domain you will use (leave empty to run locally on http://localhost): " domain
    domain="${domain#http://}"
    domain="${domain#https://}"
    domain="${domain%/}"

    if [ -z "$domain" ]; then
      SITE_ADDRESS="http://localhost"
      SITE_SUMMARY="run locally on http://localhost"
      return
    fi

    if [[ "$domain" =~ ^[A-Za-z0-9.-]+$ ]]; then
      SITE_ADDRESS="$domain"
      SITE_SUMMARY="serve https://$domain with an automatic Let's Encrypt certificate"
      return
    fi

    info "That does not look like a valid domain. Use letters, digits, dots and hyphens."
  done
}

ask_password() {
  local answer pw
  read -rp "Generate a strong random database password automatically? [Y/n] " answer

  if [[ "$answer" =~ ^[Nn]$ ]]; then
    while true; do
      read -rsp "Enter your PostgreSQL password: " pw; echo
      [ -n "$pw" ] && break
      info "Password cannot be empty."
    done
    DB_PASSWORD="$pw"
    DB_PASSWORD_SUMMARY="using the password you provided"
  else
    DB_PASSWORD="$(generate_password)"
    DB_PASSWORD_SUMMARY="generated a strong random password"
  fi
}

# --- main -------------------------------------------------------------------

main() {
  bold "Flick setup"
  echo

  [ -f "$EXAMPLE_FILE" ] || die ".env.example not found. Run this from the Flick repository."

  if [ -f "$ENV_FILE" ]; then
    local answer
    read -rp "A .env already exists. Overwrite it? [y/N] " answer
    [[ "$answer" =~ ^[Yy]$ ]] || { echo "Keeping your existing .env. Nothing changed."; exit 0; }
  fi

  echo
  ask_site_address
  echo
  ask_password

  ENV_CONTENT="$(cat "$EXAMPLE_FILE")"$'\n'
  set_env FLICK_SITE_ADDRESS "$SITE_ADDRESS"
  set_env POSTGRES_PASSWORD "$DB_PASSWORD"

  ( umask 077; printf '%s' "$ENV_CONTENT" > "$ENV_FILE" )
  chmod 600 "$ENV_FILE"

  echo
  bold "Done. Your .env is ready."
  info "Flick will $SITE_SUMMARY."
  info "Database: $DB_PASSWORD_SUMMARY."
  echo
  info "Start Flick with: make up"
}

main "$@"
