#!/usr/bin/env python3
"""Consistent SQLite backup plus immutable uploaded media. Verify before retention."""
import sqlite3, pathlib, tempfile, tarfile, datetime, os, json
root=pathlib.Path(os.environ.get('COMMO_ROOT','/opt/commo'))
target=root/'backups';target.mkdir(mode=0o700,exist_ok=True)
os.umask(0o077)
name='commo-'+datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ')+'.tar.gz'
with tempfile.TemporaryDirectory(dir=target) as tmp:
    snap=pathlib.Path(tmp)/'shop.sqlite'
    with sqlite3.connect(root/'data/shop.sqlite') as src, sqlite3.connect(snap) as dst:
        src.backup(dst)
        assert dst.execute('PRAGMA integrity_check').fetchone()[0]=='ok'
        json.loads(dst.execute("SELECT body FROM shop_state WHERE id='shop'").fetchone()[0])
    partial=target/(name+'.partial')
    with tarfile.open(partial,'w:gz') as tar:
        tar.add(snap,arcname='shop.sqlite')
        if (root/'data/media').exists():tar.add(root/'data/media',arcname='media')
    # Restore into isolated temporary files and verify DB/media bytes.
    with tarfile.open(partial) as tar:
        tar.extractall(pathlib.Path(tmp)/'restore',filter='data')
    with sqlite3.connect(pathlib.Path(tmp)/'restore/shop.sqlite') as restored:
        assert restored.execute('PRAGMA integrity_check').fetchone()[0]=='ok'
    if (root/'data/media').exists():
        for file in (root/'data/media').iterdir():
            # Files created after snapshot are harmless; only check archived files.
            copy=pathlib.Path(tmp)/'restore/media'/file.name
            if copy.exists():assert file.read_bytes()==copy.read_bytes()
    partial.rename(target/name)
for old in sorted(target.glob('commo-*.tar.gz'))[:-14]:old.unlink()
print(name+' verified (database and media)')
