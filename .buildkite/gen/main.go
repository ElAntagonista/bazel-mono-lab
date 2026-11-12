// Command gen prints the Buildkite steps for the Bazel targets affected by this commit.
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	bk "github.com/buildkite/buildkite-sdk/sdk/go/sdk/buildkite"
)

// Changes here can affect every target without showing up in the graph.
var fullBuildTriggers = []string{".bazelrc", ".bazelversion", "MODULE.bazel", ".buildkite/"}

func run(ok []int, name string, args ...string) []string {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		code := -1
		if ee, isExit := err.(*exec.ExitError); isExit {
			code = ee.ExitCode()
		}
		accepted := false
		for _, c := range ok {
			accepted = accepted || c == code
		}
		if !accepted {
			log.Fatalf("%s %s: %v", name, strings.Join(args, " "), err)
		}
	}
	return strings.Fields(string(out))
}

func changedFiles() []string {
	ref := "HEAD~1"
	if base := os.Getenv("BUILDKITE_PULL_REQUEST_BASE_BRANCH"); base != "" && base != "false" {
		// The agent only fetches this build's ref, so origin/<base> may be stale.
		run([]int{0}, "git", "fetch", "--quiet", "origin", base)
		ref = run([]int{0}, "git", "merge-base", "origin/"+base, "HEAD")[0]
	}
	return run([]int{0}, "git", "diff", "--name-only", ref, "HEAD")
}

// query tolerates exit code 3 (partial result, e.g. a deleted file).
func query(expr string) []string {
	return run([]int{0, 3}, "bazel", "query", expr, "--keep_going")
}

func affected(files []string) (targets, tests []string, reason string) {
	for _, f := range files {
		for _, t := range fullBuildTriggers {
			if strings.HasPrefix(f, t) {
				return []string{"//..."}, query("tests(//...)"), "full build (config changed)"
			}
		}
	}
	var srcs []string
	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			srcs = append(srcs, f)
		}
	}
	if len(srcs) == 0 {
		return nil, nil, "nothing affected"
	}
	universe := fmt.Sprintf("rdeps(//..., set(%s))", strings.Join(srcs, " "))
	return query("kind(rule, " + universe + ")"), query("tests(" + universe + ")"),
		fmt.Sprintf("%d changed files", len(srcs))
}

func step(label, key, dependsOn, command string) bk.CommandStep {
	s := bk.CommandStep{
		Label:   bk.Value(label),
		Command: &bk.CommandStepCommand{String: bk.Value(command)},
		Agents:  &bk.Agents{AgentsObject: &bk.AgentsObject{"queue": "default"}},
	}
	if key != "" {
		s.Key = bk.Value(key)
	}
	if dependsOn != "" {
		s.DependsOn = &bk.DependsOn{String: bk.Value(dependsOn)}
	}
	return s
}

// buildPipeline is pure: unit-test it with fake target lists.
func buildPipeline(targets, tests []string, reason string) *bk.Pipeline {
	p := &bk.Pipeline{}
	p.AddStep(step(":mag: plan: "+reason, "", "", "echo planned"))
	if len(targets) > 0 {
		p.AddStep(step(":bazel: build", "build", "", "bazel build "+strings.Join(targets, " ")))
	}
	if len(tests) > 0 {
		p.AddStep(step(":bazel: test", "", "build", "bazel test --test_output=errors "+strings.Join(tests, " ")))
	}
	return p
}

func main() {
	// bazel run starts us in the runfiles dir; work from the repo root instead.
	if ws := os.Getenv("BUILD_WORKSPACE_DIRECTORY"); ws != "" {
		if err := os.Chdir(ws); err != nil {
			log.Fatal(err)
		}
	}
	out, err := buildPipeline(affected(changedFiles())).ToJSON()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
}
