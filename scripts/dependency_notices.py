#!/usr/bin/env python3
"""Collect exact module license/notice text for binary distributions."""
import json
from pathlib import Path
import subprocess
import argparse

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("destination", type=Path)
    target = parser.parse_args().destination.resolve()
    module = Path(__file__).resolve().parents[1]
    target.mkdir(parents=True, exist_ok=True)
    subprocess.run(['go', 'mod', 'download', 'all'], cwd=module, check=True)
    data = subprocess.check_output(['go', 'list', '-m', '-json', 'all'], cwd=module, text=True)
    decoder, pos, count = json.JSONDecoder(), 0, 0
    while pos < len(data):
        while pos < len(data) and data[pos].isspace(): pos += 1
        if pos == len(data): break
        item, pos = decoder.raw_decode(data, pos)
        if item.get('Main'): continue
        root = Path(item['Dir'])
        files = [p for p in root.iterdir() if p.is_file() and (p.name.upper().startswith('LICENSE') or p.name.upper().startswith('NOTICE') or p.name.upper().startswith('COPYING'))]
        if not files: raise RuntimeError('No license found: ' + item['Path'])
        name = item['Path'].replace('/', '_') + '@' + item['Version'] + '.txt'
        (target/name).write_text('\n\n'.join(p.name+'\n'+p.read_text() for p in files))
        count += 1
    print('Collected dependency notices:', count)
if __name__ == '__main__': main()
