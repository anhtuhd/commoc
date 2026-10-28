#!/bin/sh
set -eu
umask 077
root=/opt/commoc
stamp=$(date -u +%Y%m%dT%H%M%SZ)
backup="$root/backups/$stamp"
mkdir -p "$backup"
docker compose --env-file "$root/release.env" -f "$root/compose.yml" exec -T db pg_dump -U commoc -d commoc -Fc > "$backup/database.dump"
tar -C "$root" -czf "$backup/media.tar.gz" media
(cd "$backup" && sha256sum database.dump media.tar.gz > SHA256SUMS)
printf 'Backup complete: %s\n' "$backup"
