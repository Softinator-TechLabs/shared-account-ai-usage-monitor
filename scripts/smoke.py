#!/usr/bin/env python3
"""Exercise synthetic enrollment, full-text archive, replay, permissions and MCP."""
import argparse
import http.cookiejar
import json
import os
from pathlib import Path
import subprocess
import tempfile
from urllib.request import Request, build_opener, HTTPCookieProcessor
from urllib.error import HTTPError


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--base-url', required=True)
    parser.add_argument('--agent-binary', required=True)
    parser.add_argument('--mcp-binary', required=True)
    parser.add_argument('--upstream', required=True)
    parser.add_argument('--upstream-token-file', required=True)
    args = parser.parse_args()
    jar = http.cookiejar.CookieJar()
    browser = build_opener(HTTPCookieProcessor(jar))
    def request(path, body=None, token=None):
        headers = {'Content-Type':'application/json','Origin':args.base_url}
        if token: headers['Authorization']='Bearer '+token
        req = Request(args.base_url+path, data=None if body is None else json.dumps(body).encode(), headers=headers)
        with browser.open(req, timeout=30) as res: return json.load(res)
    if not request('/healthz').get('demo'): raise RuntimeError('Smoke requires a synthetic demo server')
    # The caller must supply the isolated synthetic AgentsView instance; never discover profiles.
    av_token = Path(args.upstream_token_file).read_text().strip()
    with build_opener().open(Request(args.upstream+'/api/v1/sessions?include_one_shot=true&include_children=true&include_automated=true', headers={'Authorization':'Bearer '+av_token}),timeout=30) as res:
        sources=json.load(res)['sessions']
    if not sources or any('synthetic-' not in s['id'] for s in sources): raise RuntimeError('Upstream is not an exclusively synthetic fixture')
    with browser.open(Request(args.base_url+'/auth/demo',data=b'{"person":"owner"}',headers={'Content-Type':'application/json','Origin':args.base_url}),timeout=30): pass
    invite=request('/api/v1/invitations',{'person':'alice'})
    with tempfile.TemporaryDirectory(prefix='telemetry-smoke-') as tmp:
        root=Path(tmp); invite_file=root/'invitation.json';invite_file.write_text(json.dumps(invite));invite_file.chmod(0o600)
        config=root/'agent.json'
        subprocess.run([args.agent_binary,'enroll','--server',args.base_url,'--config',str(config),'--invitation-file',str(invite_file),'--device','synthetic-smoke','--ack-version',str(invite['policy']['version']),'--upstream',args.upstream,'--upstream-token-file',args.upstream_token_file],check=True,capture_output=True)
        for _ in range(2): subprocess.run([args.agent_binary,'once','--config',str(config)],check=True,capture_output=True)
        rows=request('/api/v1/activity?q=expected+login+behaviour')
        matched=[r for r in rows if r['source_ref']=='codex:synthetic-codex-001']
        assert len(matched)==1, 'replay duplicate or missing source'
        row=request('/api/v1/sessions/'+matched[0]['id'])
        prompt=row['messages'][0]['content'];assert len(prompt)==29000, 'full Unicode prompt lost'
        assert row['account_method']=='unknown', 'invented account identity'
        request('/api/v1/sessions/'+row['id']+'/reviews',{'ordinal':0,'kind':'comment','body':'Synthetic smoke: add explicit verification criteria.','actor_kind':'human'})
        run=request('/api/v1/sessions/'+row['id']+'/analysis',{'ordinal':0})
        request('/api/v1/analysis/result',{'body':'Synthetic external agent draft: source prompt preserved. Outcome evidence missing.'},run['token'])
        reviews=request('/api/v1/sessions/'+row['id']+'/reviews');assert any(r['actor_kind']=='agent' for r in reviews)
        read_token=request('/api/v1/access-token',{})['token'];token_file=root/'read-token';token_file.write_text(read_token);token_file.chmod(0o600)
        queries=[{'jsonrpc':'2.0','id':1,'method':'initialize','params':{}},{'jsonrpc':'2.0','id':2,'method':'tools/list'},{'jsonrpc':'2.0','id':3,'method':'tools/call','params':{'name':'get_session','arguments':{'id':row['id']}}}]
        result=subprocess.run([args.mcp_binary,'--server',args.base_url,'--token-file',str(token_file)],input='\n'.join(json.dumps(q) for q in queries)+'\n',text=True,capture_output=True,check=True)
        replies=[json.loads(line) for line in result.stdout.splitlines()];assert len(replies)==3
        mcp_row=json.loads(replies[2]['result']['content'][0]['text']);assert mcp_row['messages'][0]['content']==prompt
        try: request('/api/v1/invitations',{'person':'bob'},read_token)
        except HTTPError as error: assert error.code==403
        else: raise AssertionError('read token performed a write')
    print(json.dumps({'synthetic':True,'full_prompt_characters':len(prompt),'replay_idempotent':True,'human_and_agent_discussion':True,'read_only_mcp':True,'no_provider_API_key':True}))
if __name__=='__main__': main()
