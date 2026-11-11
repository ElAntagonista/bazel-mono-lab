# bazel-mono-lab

A small monorepo for learning Bazel and Buildkite hands-on.

```
mono/
├── MODULE.bazel        # external dependencies: rules_python, rules_go
├── libs/greeting/      # py_library + py_test (+ testdata/names.txt)
├── services/api/       # py_binary that depends on //libs/greeting
└── tools/cli/          # go_binary, independent of the Python code
```

## Quick start

```bash
bazel build //...
bazel test //...
bazel run //services/api
bazel run //tools/cli
```

Which targets does a change to the shared library affect?

```bash
bazel query 'rdeps(//..., //libs/greeting:greeting.py)'
```
