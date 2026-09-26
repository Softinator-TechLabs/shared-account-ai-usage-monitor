import importlib.util
import json
import pathlib
import threading
import unittest
from urllib.parse import urlparse, parse_qs
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

module_path = pathlib.Path(__file__).parents[1] / 'scripts/probe_agentsview.py'
spec = importlib.util.spec_from_file_location('probe', module_path)
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)

class ProbeTests(unittest.TestCase):
    def setUp(self):
        self.secret = 'fixture-token-only'
        self.messages = [{'ordinal': 0, 'role': 'user', 'content': 'Hinglish में समझाओ ' * 1500, 'vendor_extension': {'kept': True}}]
        outer = self
        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *_): pass
            def do_GET(self):
                if self.headers.get('Authorization') != 'Bearer '+outer.secret:
                    self.send_error(401); return
                if self.path.startswith('/api/v1/sessions/demo/messages'):
                    start=int(parse_qs(urlparse(self.path).query).get('from',['0'])[0])
                    body = {'messages': outer.messages[start:start+100], 'count':len(outer.messages[start:start+100])}
                elif self.path.startswith('/api/v1/sessions?'):
                    if not all(flag in self.path for flag in ['include_one_shot=true','include_automated=true','include_children=true']):
                        self.send_error(400); return
                    body = {'sessions':[{'id':'demo','agent':'codex'}], 'total':1}
                else:
                    self.send_error(404); return
                data=json.dumps(body).encode(); self.send_response(200); self.end_headers();self.wfile.write(data)
        self.server=ThreadingHTTPServer(('127.0.0.1',0),Handler)
        threading.Thread(target=self.server.serve_forever,daemon=True).start()
        self.url=f'http://127.0.0.1:{self.server.server_port}'
    def tearDown(self): self.server.shutdown();self.server.server_close()
    def test_additive_fields_preserved(self):
        result=m.probe(self.url, token=self.secret)
        self.assertTrue(result['compatible'])
        self.assertEqual(result['records'][0]['messages'][0],self.messages[0])
    def test_count_only_pagination_collects_every_message(self):
        self.messages=[{'ordinal':n,'role':'user','content':str(n)} for n in range(201)]
        self.assertEqual(len(m.probe(self.url,token=self.secret)['records'][0]['messages']),201)
    def test_missing_content_fails(self):
        self.messages[0].pop('content')
        self.assertFalse(m.probe(self.url,token=self.secret)['compatible'])
    def test_nonloopback_rejected(self):
        with self.assertRaises(ValueError): m.probe('http://example.org',token=self.secret)
    def test_secret_not_logged(self):
        result=m.probe(self.url,token='wrong-fixture-token')
        self.assertFalse(result['compatible'])
        self.assertNotIn('wrong-fixture-token', json.dumps(result))
