#!/usr/bin/env python3
"""Export only committed product files, never local state or private parent history."""
import argparse
import hashlib
import io
import json
from pathlib import Path, PurePosixPath
import subprocess
import tarfile

ROOT = Path(__file__).resolve().parents[1]
PRIVATE = {'.git', '.local', 'node_modules', 'test-results', 'playwright-report', '__pycache__', '.DS_Store', 'team-agent.json', 'team-access-token.txt', 'device-invitation.json', 'owner-link.txt', 'auth.json', 'credentials.json'}
def export(destination: Path):
    def git(*args):
        return subprocess.check_output(['git', *args], cwd=ROOT)
    commit = git('rev-parse', 'HEAD').decode().strip()
    prefix = git('rev-parse', '--show-prefix').decode().strip().rstrip('/')
    tree = commit + (':' + prefix if prefix else '')
    # Fixed commit, not the index or working filesystem. Unknown local files cannot enter.
    top = git('rev-parse', '--show-toplevel').decode().strip()
    committed = subprocess.check_output(['git', '-C', top, 'archive', '--format=tar', tree])
    manifest = []
    with tarfile.open(fileobj=io.BytesIO(committed), mode='r:') as source, tarfile.open(destination, 'w:gz') as target:
        for member in source.getmembers():
            rel = PurePosixPath(member.name)
            if member.isdir():
                continue
            if not member.isfile() or rel.is_absolute() or '..' in rel.parts:
                raise ValueError('Non-regular or unsafe committed entry: ' + member.name)
            if any(part in PRIVATE or part.endswith(('.queue', '.lock', '.suppressed', '.checkpoint.json')) for part in rel.parts) or (rel.name.startswith('.env') and rel.name != '.env.example') or rel.name.startswith('private-') or rel.suffix in {'.sqlite', '.db', '.pem', '.key', '.log', '.pyc'}:
                raise ValueError('Private/runtime file is tracked: ' + member.name)
            data = source.extractfile(member).read()
            entry = tarfile.TarInfo(str(PurePosixPath('shared-account-ai-usage-monitor') / rel))
            entry.size = len(data); entry.mode = member.mode; entry.mtime = member.mtime
            target.addfile(entry, io.BytesIO(data))
            manifest.append({'path': str(rel), 'sha256': hashlib.sha256(data).hexdigest()})
    destination.with_suffix('.manifest.json').write_text(json.dumps({'source_commit': commit, 'files': manifest}, indent=2) + '\n')
    print(f'Exported {len(manifest)} committed product files. Local state and private repository history excluded.')

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('destination', type=Path)
    args = parser.parse_args()
    if args.destination.exists(): parser.error('destination exists; choose a new artifact name')
    args.destination.parent.mkdir(parents=True, exist_ok=True)
    export(args.destination)
