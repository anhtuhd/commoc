#!/usr/bin/env python3
"""Private off-VPS backup on the owner's configured Mac. No credentials in source."""
import subprocess,pathlib,os,re,tarfile,sqlite3,tempfile
os.umask(0o077)
root=pathlib.Path(__file__).resolve().parent.parent
out=root/'backups'/'vps';out.mkdir(parents=True,exist_ok=True,mode=0o700)
config=pathlib.Path.home()/'.codex/vps/managed-vps/config'
ssh=['ssh','-F',str(config),'-o','BatchMode=yes','-o','ConnectTimeout=20','managed-vps']
listing=subprocess.check_output(ssh+['sudo -n find /opt/commo/backups -maxdepth 1 -name "commo-*.tar.gz" -printf "%f\\n"'],text=True)
names=sorted(n for n in listing.splitlines() if re.fullmatch(r'commo-\d{8}T\d{6}Z\.tar\.gz',n))
if not names:raise SystemExit('No verified backup available')
name=names[-1];dest=out/name
if not dest.exists():
    partial=out/(name+'.partial')
    with partial.open('wb') as f:subprocess.run(ssh+['sudo -n cat /opt/commo/backups/'+name],stdout=f,check=True)
    with tempfile.TemporaryDirectory(dir=out) as tmp:
        with tarfile.open(partial) as tar:tar.extractall(tmp,filter='data')
        with sqlite3.connect(pathlib.Path(tmp)/'shop.sqlite') as db:assert db.execute('PRAGMA integrity_check').fetchone()[0]=='ok'
    partial.rename(dest)
for old in sorted(out.glob('commo-*.tar.gz'))[:-14]:old.unlink()
print('Off-VPS backup verified:',name)
