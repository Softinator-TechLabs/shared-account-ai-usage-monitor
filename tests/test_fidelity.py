import importlib.util
from pathlib import Path
import unittest
s=importlib.util.spec_from_file_location('fidelity',Path(__file__).parents[1]/'scripts/compare_fidelity.py');m=importlib.util.module_from_spec(s);s.loader.exec_module(m)
class FidelityTests(unittest.TestCase):
 def test_hinglish_long_tool_roundtrip(self):
  value={'messages':[{'role':'user','content':'समझाओ Hinglish '*2000},{'role':'tool','content':'output '*2000}]}
  self.assertEqual(m.compare(value,value)['state'],'complete')
 def test_missing_tool_output_is_partial(self):
  self.assertEqual(m.compare({'messages':[{'role':'tool','content':'required'}]},{'messages':[]})['state'],'partial')
 def test_same_text_new_prompt_is_distinct(self):
  a={'messages':[{'content':'same'},{'content':'same'}]}
  self.assertEqual(m.compare(a,{'messages':[{'content':'same'}]})['missing_records'],1)
 def test_account_not_backfilled(self):
  self.assertIsNone(m.compare({'messages':[]},{'messages':[],'current_account':'new'})['historical_account'])
 def test_fixture_only_not_device_verified(self):
  self.assertEqual(m.evaluate_support([{'os':'macos','fixture_pass':True}])['eligible'],[])
 def test_partial_antigravity_not_full(self):
  self.assertEqual(m.evaluate_support([{'os':'macos','fixture_pass':True,'device_pass':True,'fidelity':'partial'}])['eligible'],[])
