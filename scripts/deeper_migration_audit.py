#!/usr/bin/env python3
import re
import glob
import json

MIG_GLOB = 'migrations/*.sql'

files = sorted(glob.glob(MIG_GLOB))

create_tbl_re = re.compile(r'CREATE TABLE(?: IF NOT EXISTS)?\s+([a-zA-Z0-9_]+)\s*\((.*?)\);', re.S | re.I)
create_index_re = re.compile(r'CREATE\s+(UNIQUE\s+)?INDEX(?: IF NOT EXISTS)?\s+[a-zA-Z0-9_.]+\s+ON\s+([a-zA-Z0-9_]+)\s*\(([^)]+)\)', re.I)
insert_re = re.compile(r'INSERT\s+INTO\s+([a-zA-Z0-9_]+)\s*(\(([^)]*)\))?', re.I)
on_conflict_re = re.compile(r'ON\s+CONFLICT\s*\(([^)]+)\)', re.I)

def normalize_ident(s):
    return s.strip().strip('"').lower()

# Collect table metadata: not-null columns (without DEFAULT), unique constraints
tables_notnull = {}
unique_constraints = {}

for path in files:
    txt = open(path, 'r', encoding='utf-8').read()
    for m in create_tbl_re.finditer(txt):
        tbl = normalize_ident(m.group(1))
        cols_blob = m.group(2)
        # naive column split
        col_lines = [c.strip() for c in re.split(r',\n', cols_blob)]
        notnulls = set()
        # find UNIQUE constraints inside table
        uqcols = []
        for line in col_lines:
            # column defs like: name TYPE NOT NULL DEFAULT ...
            col_match = re.match(r'\s*"?([a-zA-Z0-9_]+)"?\s+(.+)', line)
            if col_match:
                name = normalize_ident(col_match.group(1))
                rest = col_match.group(2).upper()
                if 'NOT NULL' in rest and 'DEFAULT' not in rest:
                    notnulls.add(name)
            # table-level UNIQUE (name1, name2)
            uq = re.match(r'.*UNIQUE\s*\(([^)]+)\)', line, re.I)
            if uq:
                cols = [normalize_ident(c) for c in uq.group(1).split(',')]
                uqcols.append(tuple(cols))
        if notnulls:
            tables_notnull.setdefault(tbl, set()).update(notnulls)
        if uqcols:
            unique_constraints.setdefault(tbl, set()).update(uqcols)

    for m in create_index_re.finditer(txt):
        is_unique = bool(m.group(1))
        tbl = normalize_ident(m.group(2))
        cols = [normalize_ident(c) for c in m.group(3).split(',')]
        if is_unique:
            unique_constraints.setdefault(tbl, set()).add(tuple(cols))

# Find DO $$ blocks ranges for guarding analysis

def find_do_blocks(txt):
    blocks = []
    for m in re.finditer(r'DO\s+\$\$.*?\$\$;', txt, re.S | re.I):
        blocks.append((m.start(), m.end(), m.group(0)))
    return blocks

issues = []
conflict_issues = []

for path in files:
    txt = open(path, 'r', encoding='utf-8').read()
    lowered = txt.lower()
    do_blocks = find_do_blocks(txt)

    # map inserts
    for m in insert_re.finditer(txt):
        tbl = normalize_ident(m.group(1))
        cols_group = m.group(3)
        cols = None
        if cols_group:
            cols = [normalize_ident(c) for c in cols_group.split(',')]
        insert_pos = m.start()
        guarded = False
        guard_reasons = []
        # check if inside a DO block and whether that block contains guards for table/columns
        for (bstart, bend, block) in do_blocks:
            if bstart <= insert_pos <= bend:
                blk = block.lower()
                # heuristics: IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'label_templates' AND column_name = 'template_json')
                if (f"information_schema.columns" in blk or "information_schema.tables" in blk) and (tbl in blk):
                    # prefer more specific column checks
                    if 'column_name' in blk and cols is not None:
                        # check for any of the columns present in the block
                        for c in cols:
                            if f"column_name = '{c}'" in blk or f'column_name = "{c}"' in blk:
                                guarded = True
                                guard_reasons.append(f"DO-block checks for column {c}")
                    # if block checks table existence, treat as guarded for general presence
                    if f"table_name = '{tbl}'" in blk or f'"{tbl}"' in blk:
                        guarded = True
                        guard_reasons.append('DO-block checks for table existence')
                # other heuristics: IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'x')
                if re.search(r"information_schema\.tables.*table_name\s*=\s*'{}'".format(tbl), blk):
                    guarded = True
                    guard_reasons.append('DO-block table guard')
                # if block uses IF EXISTS (SELECT 1 FROM information_schema.columns WHERE column_name = '...')
                # It's good enough.
        # now check NOT NULL columns absent in insert
        if tbl in tables_notnull:
            notnulls = tables_notnull[tbl]
            if cols is not None:
                missing = notnulls - set(cols)
            else:
                missing = notnulls.copy()
            if missing and not guarded:
                issues.append({
                    'file': path,
                    'table': tbl,
                    'insert_columns': cols,
                    'missing_notnull_columns': sorted(list(missing)),
                    'guarded': guarded,
                    'guard_reasons': guard_reasons,
                })
    # ON CONFLICT checks
    for m in on_conflict_re.finditer(txt):
        cols = [normalize_ident(c) for c in m.group(1).split(',')]
        # find preceding INSERT INTO table near this ON CONFLICT
        before = txt[:m.start()]
        ins = list(insert_re.finditer(before))
        if ins:
            last_insert = ins[-1]
            tbl = normalize_ident(last_insert.group(1))
            # check if unique constraint exists for same column set
            found = False
            uq_sets = unique_constraints.get(tbl, set())
            for uq in uq_sets:
                if tuple(cols) == tuple(uq):
                    found = True
                    break
            if not found:
                conflict_issues.append({
                    'file': path,
                    'table': tbl,
                    'on_conflict_columns': cols,
                    'known_unique_constraints': [list(u) for u in uq_sets],
                })

# Print results
out = {
    'issues_missing_notnull_without_guard': issues,
    'on_conflict_without_unique': conflict_issues,
}
print(json.dumps(out, indent=2))
