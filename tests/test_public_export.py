import importlib.util
import subprocess
import tarfile
import tempfile
import unittest
from pathlib import Path

spec=importlib.util.spec_from_file_location('public_export',Path(__file__).parents[1]/'scripts/export_public.py')
module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
class PublicExportTests(unittest.TestCase):
 def test_only_tracked_product_files_can_be_exported(self):
  with tempfile.TemporaryDirectory() as temp:
   root=Path(temp)/'source';root.mkdir();subprocess.run(['git','init','-q',str(root)],check=True)
   (root/'README.md').write_text('Synthetic public product')
   (root/'team-agent.json').write_text('{"token":"synthetic-private"}')
   (root/'anything-private.txt').write_text('Synthetic private prompt')
   queue=root/'team-agent.json.queue';queue.mkdir();(queue/'payload.json').write_text('Synthetic transcript')
   subprocess.run(['git','-C',str(root),'add','README.md'],check=True)
   subprocess.run(['git','-C',str(root),'-c','user.name=Synthetic','-c','user.email=synthetic@example.com','commit','-qm','public fixture'],check=True)
   original=module.ROOT;module.ROOT=root
   try:
    archive=Path(temp)/'release.tar.gz';module.export(archive)
    with tarfile.open(archive) as f:names=f.getnames()
    self.assertEqual(names,['shared-account-ai-usage-monitor/README.md'])
   finally:module.ROOT=original
 def test_nested_product_exports_only_its_committed_tree(self):
  with tempfile.TemporaryDirectory() as temp:
   repo=Path(temp)/'private';repo.mkdir();subprocess.run(['git','init','-q',str(repo)],check=True)
   product=repo/'product';product.mkdir();(product/'README.md').write_text('Synthetic public source');(repo/'private-prompt.txt').write_text('Synthetic private evidence')
   subprocess.run(['git','-C',str(repo),'add','.'],check=True);subprocess.run(['git','-C',str(repo),'-c','user.name=Synthetic','-c','user.email=synthetic@example.com','commit','-qm','fixture'],check=True)
   (product/'README.md').write_text('Uncommitted text must not ship')
   original=module.ROOT;module.ROOT=product
   try:
    archive=Path(temp)/'release.tar.gz';module.export(archive)
    with tarfile.open(archive) as f:
     self.assertEqual(f.getnames(),['shared-account-ai-usage-monitor/README.md'])
     self.assertEqual(f.extractfile(f.getmembers()[0]).read(),b'Synthetic public source')
   finally:module.ROOT=original
