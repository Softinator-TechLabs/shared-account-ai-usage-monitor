#!/usr/bin/env python3
"""Read-only, loopback-only AgentsView capability probe; never discovers credentials."""
import argparse
import ipaddress
import json
from pathlib import Path
from urllib.parse import urlparse, urlencode, quote
from urllib.request import Request, build_opener, HTTPRedirectHandler
from urllib.error import HTTPError

class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, *_):
        raise ValueError('upstream redirect refused')

def probe(base_url, token_file=None, *, token=None):
    url=urlparse(base_url)
    try:
        local=ipaddress.ip_address(url.hostname or '').is_loopback
    except ValueError:
        local=False
    if url.scheme not in ('http','https') or not local or url.username or url.password or url.query or url.fragment:
        raise ValueError('explicit loopback IP endpoint required')
    if token_file:
        token=Path(token_file).read_text().strip()
    opener=build_opener(NoRedirect())
    def get(path, params=None):
        endpoint=base_url.rstrip('/')+path+('?' + urlencode(params) if params else '')
        request=Request(endpoint,headers={'Authorization':'Bearer '+token} if token else {})
        with opener.open(request,timeout=10) as response:
            value=json.load(response)
        if not isinstance(value,dict):raise ValueError('expected JSON object')
        return value
    records=[]
    result={'compatible':False,'records':records,'errors':[]}
    try:
        cursor=''; seen=set()
        while True:
            params={'include_one_shot':'true','include_automated':'true','include_children':'true','include_source':'true','limit':100}
            if cursor:params['cursor']=cursor
            page=get('/api/v1/sessions',params)
            if not isinstance(page.get('sessions'),list):raise ValueError('sessions field unavailable')
            for session in page['sessions']:
                sid=session.get('id')
                if not isinstance(sid,str):raise ValueError('session ID unavailable')
                messages=[]; start=0
                while True:
                    mp=get('/api/v1/sessions/'+quote(sid,safe='')+'/messages',{'from':start,'limit':100,'direction':'asc'})
                    batch=mp.get('messages')
                    if not isinstance(batch,list):raise ValueError('message enumeration unavailable')
                    for msg in batch:
                        if not isinstance(msg.get('content'),str) or not isinstance(msg.get('ordinal'),int):
                            raise ValueError('message content/ordinal unavailable')
                    messages.extend(batch)
                    more=mp.get('has_more',False) or len(messages)<mp.get('total',len(messages)) or len(batch)==100
                    if not more:break
                    if not batch or batch[-1]['ordinal']<start:raise ValueError('message pagination stalled')
                    start=batch[-1]['ordinal']+1
                records.append({'session':session,'messages':messages})
            cursor=page.get('next_cursor')
            if not cursor:break
            if cursor in seen:raise ValueError('session pagination stalled')
            seen.add(cursor)
        if not records or not any(r['messages'] for r in records):raise ValueError('no fixture content to prove capability')
        result['compatible']=True
    except HTTPError as exc:
        result['errors'].append('upstream HTTP '+str(exc.code))
    except Exception as exc:
        # URLs, credentials and bodies never appear in diagnostics.
        result['errors'].append(str(exc) if isinstance(exc,ValueError) else type(exc).__name__)
    return result

if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base-url',required=True);parser.add_argument('--token-file')
    args=parser.parse_args(); result=probe(args.base_url,args.token_file)
    print(json.dumps({k:v for k,v in result.items() if k!='records'},indent=2))
    raise SystemExit(0 if result['compatible'] else 1)
