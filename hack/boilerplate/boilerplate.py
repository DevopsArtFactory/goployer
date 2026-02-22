#!/usr/bin/env python
from __future__ import print_function

import argparse
import glob
import os
import re
import sys


SKIPPED_DIRS = ["themes", "Godeps", "third_party", ".git", "vendor", "examples", "base-infrastructure", "node_modules", "venv"]
SKIPPED_FILES = ["install_golint.sh", "boilerplate.py", "golangci-lint.sh", "pedantic-imports.sh", "preview-docs.sh", "gofmt.sh", "boilerplate.sh", "linters.sh", "deploy.sh", "test_userdata.sh"]

parser = argparse.ArgumentParser()
parser.add_argument("filenames", help="list of files to check, all files if unspecified", nargs='*')

rootdir = os.path.dirname(__file__) + "/../"
rootdir = os.path.abspath(rootdir)
parser.add_argument("--rootdir", default=rootdir, help="root directory to examine")

default_boilerplate_dir = os.path.join(rootdir, "/boilerplate")
parser.add_argument("--boilerplate-dir", default=default_boilerplate_dir)
args = parser.parse_args()


def get_refs():
    refs = {}

    for path in glob.glob(os.path.join(args.boilerplate_dir, "boilerplate.*.txt")):
        extension = os.path.basename(path).split(".")[1]

        ref_file = open(path, 'r')
        ref = ref_file.read().splitlines()
        ref_file.close()
        refs[extension] = ref

    return refs

def file_passes(filename, refs, regexs):
    try:
        f = open(filename, 'r')
    except:
        return False

    data = f.read()
    f.close()

    basename = os.path.basename(filename)
    extension = file_extension(filename)
    if extension != "":
        ref = refs[extension]
    else:
        ref = refs[basename]

    # remove build tags from the top of Go files
    if extension == "go":
        p = regexs["go_build_constraints"]
        (data, found) = p.subn("", data, 1)

    # remove shebang from the top of shell files
    elif extension == "sh":
        p = regexs["shebang"]
        (data, found) = p.subn("", data, 1)

    data = data.splitlines()

    # if our test file is smaller than the reference it surely fails!
    if len(ref) > len(data):
        return False

    # trim our file to the same number of lines as the reference file
    data = data[:len(ref)]

    p = regexs["year"]
    for d in data:
        if p.search(d):
            return False

    # Replace all occurrences of the regex "2017|2016|2015|2014" with "YEAR"
    p = regexs["date"]
    for i, d in enumerate(data):
        (data[i], found) = p.subn('YEAR', d)
        if found != 0:
            break

    # if we don't match the reference at this point, fail
    if ref != data:
        return False

    return True

def file_extension(filename):
    return os.path.splitext(filename)[1].split(".")[-1].lower()

def normalize_files(files):
    newfiles = []
    for i, pathname in enumerate(files):
        if not os.path.isabs(pathname):
            newfiles.append(os.path.join(args.rootdir, pathname))
    return newfiles

def get_files(extensions):
    files = []
    if len(args.filenames) > 0:
        files = args.filenames
    else:
        for root, dirs, walkfiles in os.walk(args.rootdir):
            for d in SKIPPED_DIRS:
                if d in dirs:
                    dirs.remove(d)

            for name in walkfiles:
                if name not in SKIPPED_FILES:
                    pathname = os.path.join(root, name)
                    files.append(pathname)

    files = normalize_files(files)
    outfiles = []
    for pathname in files:
        basename = os.path.basename(pathname)
        extension = file_extension(pathname)
        if extension in extensions or basename in extensions:
            outfiles.append(pathname)
    return outfiles

def get_regexs():
    regexs = {}
    # Search for "YEAR" which exists in the boilerplate, but shouldn't in the real thing
    regexs["year"] = re.compile( 'YEAR' )
    # dates can be 2019, company holder names can be anything
    regexs["date"] = re.compile(r"\b(19|20)\d{2}\b")
    # strip // +build \n\n build constraints
    regexs["go_build_constraints"] = re.compile(r"^(// \+build.*\n)+\n", re.MULTILINE)
    # strip #!.* from shell scripts
    regexs["shebang"] = re.compile(r"^(#!.*\n)\n*", re.MULTILINE)
    return regexs

def main():
    regexs = get_regexs()
    refs = get_refs()
    filenames = get_files(refs.keys())

    for filename in filenames:
        if not file_passes(filename, refs, regexs):
            print(filename, file=sys.stdout)

if __name__ == "__main__":
  sys.exit(main())
