#!/bin/python

import contextlib
import os
import shutil
from six.moves import urllib
import subprocess
import sys
import tarfile
import urllib2
import yaml

script_dir = os.path.dirname(os.path.realpath(__file__))
target_dir = os.path.join("/", "usr","share","collectd") if len(sys.argv) < 2 else sys.argv[1]
python_executable = sys.executable if sys.executable else 'python'

with file(os.path.join(script_dir, "..", "collectd-plugins.yaml"), 'r') as f:
    plugins = yaml.safe_load(f)

for p in plugins:
    plugin_name = p.get('name')
    version = p.get('version')
    repo = p.get('repo')
    url = "https://github.com/{repo}/archive/{version}.tar.gz".format(repo=repo, version=version)

    print("""Bundling...
plugin:  {p} 
version: {v} 
repo:    {r}
url:     {u}""".format(p=plugin_name, v=version, r=repo, u=url))

    with contextlib.closing(urllib.request.urlopen(url)) as stream:
        with tarfile.open(fileobj=stream, mode='r|gz') as tar_archive:
            tar_archive.extractall(target_dir)
            plugin_dir = os.path.join(target_dir, plugin_name)
            os.rename(os.path.join(target_dir, tar_archive.getnames()[0]),
                      plugin_dir)

    # install pip deps
    for package in p.get('pip_packages', []):
        subprocess.check_call([python_executable, '-m', 'pip', 'install', '-qq', '--no-warn-script-location', package])

    requirements_file = os.path.join(plugin_dir, "requirements.txt")
    if os.path.isfile(requirements_file):
        subprocess.check_call([python_executable, '-m', 'pip', 'install', '-qq', '--no-warn-script-location', '-r', requirements_file])


    # remove unecessary things
    for elem in p.get('can_remove', []):
        def rmtree_error_handler(*args):
            print("unable to remove element {0}".format(elem))
        shutil.rmtree(os.path.join(plugin_dir, elem), onerror=rmtree_error_handler)
