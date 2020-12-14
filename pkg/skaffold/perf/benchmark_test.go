package perf

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var testProj = flag.String("target", "examples/getting-started", "The target skaffold project dir")
var skDir = flag.String("dir", ".", "Skaffold root dir")

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func BenchmarkRender(b *testing.B) {
	skRoot, err := filepath.Abs(*skDir)
	if err != nil {
		b.Fatalf("failed to process path: %v", err)
	}
	cmd := exec.Command(filepath.Join(skRoot, "out/skaffold"), "render")

	cmd.Dir = filepath.Join(skRoot, *testProj)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	for i := 0; i < b.N; i++ {
		err := cmd.Run()
		if err != nil {
			b.Errorf("failed to run skaffold: %v", err)
		}
	}
}

func BenchmarkBuild(b *testing.B) {
	skRoot, err := filepath.Abs(*skDir)
	if err != nil {
		b.Fatalf("failed to process path: %v", err)
	}
	cmd := exec.Command(filepath.Join(skRoot, "out/skaffold"), "build")

	cmd.Dir = filepath.Join(skRoot, *testProj)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	for i := 0; i < b.N; i++ {
		err := cmd.Run()
		if err != nil {
			b.Errorf("failed to run skaffold: %v", err)
		}
	}
}

func BenchmarkDeploy(b *testing.B) {
	skRoot, err := filepath.Abs(*skDir)
	if err != nil {
		b.Fatalf("failed to process path: %v", err)
	}
	cmd := exec.Command(filepath.Join(skRoot, "out/skaffold"), "deploy", "-t", "foo")

	cmd.Dir = filepath.Join(skRoot, *testProj)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	for i := 0; i < b.N; i++ {
		err := cmd.Run()
		if err != nil {
			b.Errorf("failed to run skaffold: %v", err)
		}
	}
}
