"""Compare synthetic source expectations with normalized API records without inferring identity."""
def compare(expected, observed):
    want, got = expected.get('messages', []), observed.get('messages', [])
    missing = max(0, len(want)-len(got))
    changes=[]
    for index, message in enumerate(want):
        if index >= len(got): break
        for field, value in message.items():
            if got[index].get(field) != value:
                changes.append({'ordinal':index,'field':field})
    return {'state':'complete' if want == got else 'partial',
            'missing_records':missing,'changed_fields':changes,
            'historical_account':None,'historical_actor':None}

def evaluate_support(rows):
    eligible=[r for r in rows if r.get('fixture_pass') and r.get('device_pass') and r.get('fidelity')=='complete']
    return {'eligible':eligible,'implementation_gate':'pass' if eligible else 'blocked'}
