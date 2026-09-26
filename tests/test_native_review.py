import importlib.util
from pathlib import Path
import unittest
spec=importlib.util.spec_from_file_location('runner',Path(__file__).parents[1]/'scripts/run_review.py')
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
class RunnerTests(unittest.TestCase):
    def test_arguments_are_explicit_and_no_shell_interpolation(self):
        args=m.build_command('/trusted/codex','/private/output','model;touch nope')
        self.assertIn('--ignore-user-config',args)
        self.assertIn('--ephemeral',args)
        self.assertEqual(args[args.index('--sandbox')+1],'read-only')
        self.assertEqual(args[args.index('--disable')+1],'shell_tool')
        self.assertEqual(args[args.index('--model')+1],'model;touch nope')
        self.assertEqual(args[-1],'-')
