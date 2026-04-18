package main

import (
	"os"
	"testing"

	"github.com/cenkalti/work/internal/cli/agent"
	"github.com/cenkalti/work/internal/cli/task"
	"github.com/cenkalti/work/internal/cli/work"
	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"work":  func() { exit(work.Root().Execute()) },
		"agent": func() { exit(agent.Root().Execute()) },
		"task":  func() { exit(task.Root().Execute()) },
	})
}

func exit(err error) {
	if err != nil {
		os.Exit(1)
	}
}

func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/script",
		Setup: func(e *testscript.Env) error {
			e.Setenv("HOME", e.WorkDir)
			e.Setenv("WORK_PROJECTS_DIR", e.WorkDir+"/projects")
			e.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
			e.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
			e.Setenv("GIT_AUTHOR_NAME", "test")
			e.Setenv("GIT_AUTHOR_EMAIL", "t@e")
			e.Setenv("GIT_COMMITTER_NAME", "test")
			e.Setenv("GIT_COMMITTER_EMAIL", "t@e")
			return nil
		},
	})
}
