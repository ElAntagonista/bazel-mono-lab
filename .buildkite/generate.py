#!/usr/bin/env python3
"""Print Buildkite steps (JSON) for the Bazel targets affected by this commit."""
import json
import os
import subprocess

# Changes here can affect every target without showing up in the graph.
FULL_BUILD_TRIGGERS = (".bazelrc", ".bazelversion", "MODULE.bazel", ".buildkite/")


def out(*args, ok=(0,)):
    p = subprocess.run(args, capture_output=True, text=True)
    if p.returncode not in ok:
        raise SystemExit(f"{' '.join(args)} failed:\n{p.stderr}")
    return p.stdout.split()


def changed_files():
    base = os.environ.get("BUILDKITE_PULL_REQUEST_BASE_BRANCH")
    ref = out("git", "merge-base", f"origin/{base}", "HEAD")[0] if base else "HEAD~1"
    return out("git", "diff", "--name-only", ref, "HEAD")


def query(expr):
    # exit 3 = partial result (e.g. a deleted file); fine for planning.
    return out("bazel", "query", expr, "--keep_going", ok=(0, 3))


def affected(files):
    if any(f.startswith(FULL_BUILD_TRIGGERS) for f in files):
        return ["//..."], query("tests(//...)"), "full build (config changed)"
    srcs = [f for f in files if os.path.exists(f)]
    if not srcs:
        return [], [], "nothing affected"
    universe = f"rdeps(//..., set({' '.join(srcs)}))"
    return query(f"kind(rule, {universe})"), query(f"tests({universe})"), f"{len(srcs)} changed files"


def pipeline(targets, tests, reason):
    steps = [{"label": f":mag: plan: {reason}", "command": "echo planned"}]
    if targets:
        steps.append({"label": ":bazel: build", "key": "build", "agents": {"queue": "default"},
                      "command": "bazel build " + " ".join(targets)})
    if tests:
        steps.append({"label": ":bazel: test", "depends_on": "build", "agents": {"queue": "default"},
                      "command": "bazel test --test_output=errors " + " ".join(tests)})
    return {"steps": steps}


if __name__ == "__main__":
    print(json.dumps(pipeline(*affected(changed_files())), indent=2))
