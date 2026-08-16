#!/bin/sh
set -eu

install_dir=${PGSENTINEL_INSTALL_DIR:-pgsentinel}
compose_url=https://raw.githubusercontent.com/matta813/pgsentinel/main/docker-compose.quickstart.yml

command -v curl >/dev/null 2>&1 || { echo "curl is required" >&2; exit 1; }
command -v openssl >/dev/null 2>&1 || { echo "openssl is required" >&2; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "Docker is required" >&2; exit 1; }
docker compose version >/dev/null 2>&1 || { echo "Docker Compose v2 is required" >&2; exit 1; }

mkdir -p "$install_dir"
install_path=$(cd "$install_dir" && pwd)
cd "$install_path"
umask 077

curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 \
  "$compose_url" --output docker-compose.yml

if [ ! -f .env ]; then
  encryption_key=$(openssl rand -base64 32 | tr -d '\n')
  admin_password=$(openssl rand -base64 24 | tr -d '\n')
  {
    printf 'PGSENTINEL_ENCRYPTION_KEY=%s\n' "$encryption_key"
    printf 'PGSENTINEL_ADMIN_PASSWORD=%s\n' "$admin_password"
    printf 'PGSENTINEL_PORT=8080\n'
    printf 'TZ=UTC\n'
  } > .env
  chmod 600 .env
  credentials_created=true
else
  credentials_created=false
fi

docker compose pull
docker compose up -d --wait --wait-timeout 120

published_port=$(sed -n 's/^PGSENTINEL_PORT=//p' .env | tail -n 1)
published_port=${published_port:-8080}

echo
echo "PGSentinel is running at http://localhost:$published_port"
if [ "$credentials_created" = true ]; then
  echo "Administrator password: $admin_password"
  echo "The credentials are stored in $install_path/.env; keep that file private and backed up."
else
  echo "Existing credentials in $install_path/.env were preserved."
fi
