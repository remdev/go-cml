package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAndGenerate(t *testing.T) {
	root := filepath.Join("..", "examples", "ddd-sample.cml")
	var stdout, stderr bytes.Buffer
	if code := Run(&stdout, &stderr, []string{"validate", "-i", root}); code != 0 {
		t.Fatalf("validate code %d\n%s\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "has been validated without errors") {
		t.Fatal(stdout.String())
	}
	dir := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	if code := Run(&stdout, &stderr, []string{"generate", "-i", root, "-g", "plantuml", "-o", dir}); code != 0 {
		t.Fatalf("generate code %d\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Generated into '"+dir+"'.") {
		t.Fatal(stdout.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "ddd-sample_ContextMap.puml")); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	template := filepath.Join(dir, "names.ftl")
	if err := os.WriteFile(template, []byte("<#list boundedContexts as bc>${bc.name} </#list>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := Run(&stdout, &stderr, []string{"generate", "-i", root, "-g", "generic", "-o", dir, "-t", template, "-f", "names.txt"}); code != 0 {
		t.Fatalf("generic code %d\n%s", code, stderr.String())
	}
	body, err := os.ReadFile(filepath.Join(dir, "names.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "LocationContext") {
		t.Fatalf("template output %s", body)
	}
}

func TestUpstreamCLIContract(t *testing.T) {
	input := filepath.Join("..", "testdata", "cli", "test.cml")
	var stdout, stderr bytes.Buffer
	if code := Run(&stdout, &stderr, []string{"validate", "-i", input}); code != 0 {
		t.Fatalf("validate %d %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "The CML file '"+input+"' has been validated without errors.") {
		t.Fatal(stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run(&stdout, &stderr, []string{"validate", "-i", "missing.cml"}); code != 1 || !strings.Contains(stderr.String(), "ERROR: The file 'missing.cml' does not exist.") {
		t.Fatalf("missing file code %d %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	text := filepath.Join("..", "testdata", "cli", "test.txt")
	if code := Run(&stdout, &stderr, []string{"validate", "-i", text}); code != 1 || !strings.Contains(stderr.String(), "ERROR: Please provide a path to a CML (*.cml) file.") {
		t.Fatalf("non cml code %d %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	broken := filepath.Join("..", "testdata", "cli", "test-with-error.cml")
	if code := Run(&stdout, &stderr, []string{"validate", "-i", broken}); code != 1 || !strings.Contains(stderr.String(), "ERROR in ") {
		t.Fatalf("syntax code %d %s", code, stderr.String())
	}

	dir := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	if code := Run(&stdout, &stderr, []string{"generate", "-i", input, "-g", "plantuml", "-o", dir}); code != 0 {
		t.Fatal(stderr.String())
	}
	for _, name := range []string{"test_ContextMap.puml", "test_BC_CargoBookingContext.puml", "test_BC_LocationContext.puml", "test_BC_VoyagePlanningContext.puml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatal(name, err)
		}
	}
	stdout.Reset()
	if code := Run(&stdout, &stderr, []string{"generate", "-i", input, "-g", "context-map", "-o", dir}); code != 0 {
		t.Fatal(stderr.String())
	}
	for _, name := range []string{"test_ContextMap.gv", "test_ContextMap.png", "test_ContextMap.svg"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatal(name, err)
		}
	}
	stdout.Reset()
	stderr.Reset()
	template := filepath.Join("..", "testdata", "cli", "test.ftl")
	if code := Run(&stdout, &stderr, []string{"generate", "-i", input, "-g", "generic", "-o", dir, "-t", template, "-f", "test.md"}); code != 0 {
		t.Fatal(stderr.String())
	}
	body, err := os.ReadFile(filepath.Join(dir, "test.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "CargoBookingContext") {
		t.Fatalf("generic output:\n%s", body)
	}
	stderr.Reset()
	if code := Run(&stdout, &stderr, []string{"generate", "-i", input, "-g", "generic", "-o", dir, "-f", "test.md"}); code == 0 || !strings.Contains(stderr.String(), "The --template (-t) parameter is required") {
		t.Fatalf("missing template code %d %s", code, stderr.String())
	}
	stderr.Reset()
	if code := Run(&stdout, &stderr, []string{"generate", "-i", input, "-g", "plantuml", "-o", filepath.Join(dir, "nope")}); code != 1 || !strings.Contains(stderr.String(), "does not exist") {
		t.Fatalf("missing dir code %d %s", code, stderr.String())
	}
}

func TestRefactorCLI(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.cml")
	var stdout, stderr bytes.Buffer
	code := Run(&stdout, &stderr, []string{
		"refactor", "-i", filepath.Join("..", "examples", "ddd-sample.cml"),
		"-o", out, "-r", "extract-shared-kernel",
		"CargoBookingContext", "VoyagePlanningContext", "SharedVoyage",
	})
	if code != 0 {
		t.Fatalf("code %d %s", code, stderr.String())
	}
	body, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "BoundedContext SharedVoyage") {
		t.Fatal(string(body))
	}
}
