#!/usr/bin/env python3
import re
import glob

migrations = glob.glob('migrations/*.sql')
create_re = re.compile(r'CREATE TABLE(?: IF NOT EXISTS)?\s+([a-zA-Z0-9_]+)\s*\((.*?)\);', re.S | re.I)
insert_re = re.compile(r'INSERT INTO\s+([a-zA-Z0-9_]+)\s*(\(([^)]*)\))?', re.I)
notnull_re = re.compile(r'([a-zA-Z0-9_]+)\s+[A-Z0-9()_, ]+NOT NULL', re.I)

creates = {}
for path in migrations:
    s = open(path).read()
    for m in create_re.finditer(s):
        tbl = m.group(1)
        cols_blob = m.group(2)
        notnulls = [c.group(1).strip() for c in notnull_re.finditer(cols_blob)]
        if notnulls:
            creates.setdefault(tbl, set()).update(notnulls)

inserts = []
for path in migrations:
    s = open(path).read()
    for m in insert_re.finditer(s):
        tbl = m.group(1)
        cols_group = m.group(3)
        cols = None
        if cols_group:
            cols = [c.strip().strip('"') for c in cols_group.split(',')]
        inserts.append((path, tbl, cols))

warnings = []
for path, tbl, cols in inserts:
    if tbl in creates and cols is not None:
        missing = creates[tbl] - set(cols)
        if missing:
            warnings.append((path, tbl, cols, list(missing)))
    elif tbl in creates and cols is None:
        # insert without column list; warn
        warnings.append((path, tbl, None, list(creates[tbl])))

if not warnings:
    print('No obvious INSERT -> NOT NULL mismatch found (heuristic).')
else:
    print('Potential issues:')
    for p,t,c,m in warnings:
        print(f'- In {p}: INSERT into table {t} with columns {c}; NOT NULL columns defined: {m}')
