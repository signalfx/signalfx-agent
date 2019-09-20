#!/usr/bin/env python3

import csv
import glob
import pathlib
import yaml

all_metrics = ""

path = pathlib.Path.cwd() / 'all_metrics.csv'
with path.open(mode='w') as csvfile:
    fieldnames = ['monitor', 'metric', 'description', 'default', 'type']
    writer = csv.DictWriter(csvfile, fieldnames=fieldnames)
    writer.writeheader()
    for filename in glob.glob("internal/monitors/**/metadata.yaml", recursive=True):
        with open(filename) as f:
            data = yaml.safe_load(f)
            mons = data['monitors']
            if mons is None:
                continue
            for m in mons:
                if 'metrics' not in m.keys():
                    continue
                metrics = m['metrics']
                if metrics is None:
                    continue
                for mm in metrics:
                    if 'description' not in metrics[mm].keys():
                        desc = "THIS METRIC HAS NO DESCRIPTION"
                    else:
                        desc = str(metrics[mm]['description']).replace("\n", " ")
                    if desc is None:
                        desc = "THE METRIC DESCRIPTION IS BLANK"
                    if 'default' not in metrics[mm].keys():
                        default = False
                    else:
                        default = metrics[mm]['default']
                    if default is None:
                        default = "DEFAULT IS BLANK"
                    if 'type' not in metrics[mm].keys():
                        mtype = "THIS METRICS HAS NO TYPE"
                    else:
                        mtype = metrics[mm]['type']
                    if mtype is None:
                        mtype = "TYPE IS BLANK"
                    
                    writer.writerow({'monitor': m['monitorType'], 'metric': mm, 'description': desc, 'default': str(default), 'type': mtype})
    csvfile.close()