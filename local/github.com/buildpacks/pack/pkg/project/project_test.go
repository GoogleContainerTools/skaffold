package project

import (
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/buildpacks/lifecycle/api"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestProject(t *testing.T) {
	h.RequireDocker(t)
	color.Disable(true)
	defer color.Disable(false)

	spec.Run(t, "Provider", testProject, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testProject(t *testing.T, when spec.G, it spec.S) {
	var (
		logger     *logging.LogWithWriters
		readStdout func() string
	)

	it.Before(func() {
		var stdout *color.Console
		stdout, readStdout = h.MockWriterAndOutput()
		stderr, _ := h.MockWriterAndOutput()
		logger = logging.NewLogWithWriters(stdout, stderr)
	})

	when("#ReadProjectDescriptor", func() {
		it("should parse a valid v0.2 project.toml file", func() {
			projectToml := `
[_]
name = "gallant 0.2"
schema-version="0.2"
[[_.licenses]]
type = "MIT"
[_.metadata]
pipeline = "Lucerne"
[io.buildpacks]
exclude = [ "*.jar" ]
[[io.buildpacks.pre.group]]
uri = "https://example.com/buildpack/pre"
[[io.buildpacks.post.group]]
uri = "https://example.com/buildpack/post"
[[io.buildpacks.group]]
id = "example/lua"
version = "1.0"
[[io.buildpacks.group]]
uri = "https://example.com/buildpack"
[[io.buildpacks.build.env]]
name = "JAVA_OPTS"
value = "-Xmx300m"
[[io.buildpacks.env.build]]
name = "JAVA_OPTS"
value = "this-should-get-overridden-because-its-deprecated"
`
			tmpProjectToml, err := createTmpProjectTomlFile(projectToml)
			if err != nil {
				t.Fatal(err)
			}

			projectDescriptor, err := ReadProjectDescriptor(tmpProjectToml.Name(), logger)
			if err != nil {
				t.Fatal(err)
			}

			var expected string

			expected = "gallant 0.2"
			if projectDescriptor.Project.Name != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Project.Name)
			}

			expectedVersion := api.MustParse("0.2")
			if !reflect.DeepEqual(expectedVersion, projectDescriptor.SchemaVersion) {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expectedVersion, projectDescriptor.SchemaVersion)
			}

			expected = "example/lua"
			if projectDescriptor.Build.Buildpacks[0].ID != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Build.Buildpacks[0].ID)
			}

			expected = "1.0"
			if projectDescriptor.Build.Buildpacks[0].Version != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Build.Buildpacks[0].Version)
			}

			expected = "https://example.com/buildpack"
			if projectDescriptor.Build.Buildpacks[1].URI != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Build.Buildpacks[1].URI)
			}

			expected = "https://example.com/buildpack/pre"
			if projectDescriptor.Build.Pre.Buildpacks[0].URI != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Build.Pre.Buildpacks[0].URI)
			}

			expected = "https://example.com/buildpack/post"
			if projectDescriptor.Build.Post.Buildpacks[0].URI != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Build.Post.Buildpacks[0].URI)
			}

			expected = "JAVA_OPTS"
			if projectDescriptor.Build.Env[0].Name != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Build.Env[0].Name)
			}

			expected = "-Xmx300m"
			if projectDescriptor.Build.Env[0].Value != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Build.Env[0].Value)
			}

			expected = "MIT"
			if projectDescriptor.Project.Licenses[0].Type != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Project.Licenses[0].Type)
			}

			expected = "Lucerne"
			if projectDescriptor.Metadata["pipeline"] != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Metadata["pipeline"])
			}
		})

		it("should be backwards compatible with older v0.2 project.toml file", func() {
			projectToml := `
[_]
name = "gallant 0.2"
schema-version="0.2"
[[io.buildpacks.env.build]]
name = "JAVA_OPTS"
value = "-Xmx300m"
`
			tmpProjectToml, err := createTmpProjectTomlFile(projectToml)
			if err != nil {
				t.Fatal(err)
			}

			projectDescriptor, err := ReadProjectDescriptor(tmpProjectToml.Name(), logger)
			if err != nil {
				t.Fatal(err)
			}

			var expected string

			expected = "JAVA_OPTS"
			if projectDescriptor.Build.Env[0].Name != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Build.Env[0].Name)
			}

			expected = "-Xmx300m"
			if projectDescriptor.Build.Env[0].Value != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Build.Env[0].Value)
			}
		})

		it("should parse a valid v0.1 project.toml file", func() {
			projectToml := `
[project]
name = "gallant"
version = "1.0.2"
source-url = "https://github.com/buildpacks/pack"
[[project.licenses]]
type = "MIT"
[build]
exclude = [ "*.jar" ]
[[build.buildpacks]]
id = "example/lua"
version = "1.0"
[[build.buildpacks]]
uri = "https://example.com/buildpack"
[[build.env]]
name = "JAVA_OPTS"
value = "-Xmx300m"
[metadata]
pipeline = "Lucerne"
`
			tmpProjectToml, err := createTmpProjectTomlFile(projectToml)
			if err != nil {
				t.Fatal(err)
			}

			projectDescriptor, err := ReadProjectDescriptor(tmpProjectToml.Name(), logger)
			if err != nil {
				t.Fatal(err)
			}

			var expected string

			expected = "gallant"
			if projectDescriptor.Project.Name != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Project.Name)
			}

			expectedVersion := api.MustParse("0.1")
			if !reflect.DeepEqual(expectedVersion, projectDescriptor.SchemaVersion) {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expectedVersion, projectDescriptor.SchemaVersion)
			}

			expected = "1.0.2"
			if projectDescriptor.Project.Version != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Project.Version)
			}

			expected = "https://github.com/buildpacks/pack"
			if projectDescriptor.Project.SourceURL != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Project.SourceURL)
			}

			expected = "example/lua"
			if projectDescriptor.Build.Buildpacks[0].ID != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Build.Buildpacks[0].ID)
			}

			expected = "1.0"
			if projectDescriptor.Build.Buildpacks[0].Version != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Build.Buildpacks[0].Version)
			}

			expected = "https://example.com/buildpack"
			if projectDescriptor.Build.Buildpacks[1].URI != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Build.Buildpacks[1].URI)
			}

			expected = "JAVA_OPTS"
			if projectDescriptor.Build.Env[0].Name != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Build.Env[0].Name)
			}

			expected = "-Xmx300m"
			if projectDescriptor.Build.Env[0].Value != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Build.Env[0].Value)
			}

			expected = "MIT"
			if projectDescriptor.Project.Licenses[0].Type != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Project.Licenses[0].Type)
			}

			expected = "Lucerne"
			if projectDescriptor.Metadata["pipeline"] != expected {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					expected, projectDescriptor.Metadata["pipeline"])
			}
		})

		it("should create empty build ENV", func() {
			projectToml := `
[project]
name = "gallant"
`
			tmpProjectToml, err := createTmpProjectTomlFile(projectToml)
			if err != nil {
				t.Fatal(err)
			}

			projectDescriptor, err := ReadProjectDescriptor(tmpProjectToml.Name(), logger)
			if err != nil {
				t.Fatal(err)
			}

			expected := 0
			if len(projectDescriptor.Build.Env) != 0 {
				t.Fatalf("Expected\n-----\n%d\n-----\nbut got\n-----\n%d\n",
					expected, len(projectDescriptor.Build.Env))
			}

			for _, envVar := range projectDescriptor.Build.Env {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					"[]", envVar)
			}
		})

		it("should fail for an invalid project.toml path", func() {
			_, err := ReadProjectDescriptor("/path/that/does/not/exist/project.toml", logger)

			if !os.IsNotExist(err) {
				t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
					"project.toml does not exist error", "no error")
			}
		})

		it("should enforce mutual exclusivity between exclude and include", func() {
			projectToml := `
[project]
name = "bad excludes and includes"

[build]
exclude = [ "*.jar" ]
include = [ "*.jpg" ]
`
			tmpProjectToml, err := createTmpProjectTomlFile(projectToml)
			if err != nil {
				t.Fatal(err)
			}
			_, err = ReadProjectDescriptor(tmpProjectToml.Name(), logger)
			if err == nil {
				t.Fatalf(
					"Expected error for having both exclude and include defined")
			}
		})

		it("should have an id or uri defined for buildpacks", func() {
			projectToml := `
[project]
name = "missing buildpacks id and uri"

[[build.buildpacks]]
version = "1.2.3"
`
			tmpProjectToml, err := createTmpProjectTomlFile(projectToml)
			if err != nil {
				t.Fatal(err)
			}

			_, err = ReadProjectDescriptor(tmpProjectToml.Name(), logger)
			if err == nil {
				t.Fatalf("Expected error for NOT having id or uri defined for buildpacks")
			}
		})

		it("should not allow both uri and version", func() {
			projectToml := `
[project]
name = "cannot have both uri and version defined"

[[build.buildpacks]]
uri = "https://example.com/buildpack"
version = "1.2.3"
`
			tmpProjectToml, err := createTmpProjectTomlFile(projectToml)
			if err != nil {
				t.Fatal(err)
			}

			_, err = ReadProjectDescriptor(tmpProjectToml.Name(), logger)
			if err == nil {
				t.Fatal("Expected error for having both uri and version defined for a buildpack(s)")
			}
		})

		it("should require either a type or uri for licenses", func() {
			projectToml := `
[project]
name = "licenses should have either a type or uri defined"

[[project.licenses]]
`
			tmpProjectToml, err := createTmpProjectTomlFile(projectToml)
			if err != nil {
				t.Fatal(err)
			}

			_, err = ReadProjectDescriptor(tmpProjectToml.Name(), logger)
			if err == nil {
				t.Fatal("Expected error for having neither type or uri defined for licenses")
			}
		})

		it("should warn when no schema version is declared", func() {
			projectToml := ``
			tmpProjectToml, err := createTmpProjectTomlFile(projectToml)
			if err != nil {
				t.Fatal(err)
			}

			_, err = ReadProjectDescriptor(tmpProjectToml.Name(), logger)
			h.AssertNil(t, err)

			h.AssertContains(t, readStdout(), "Warning: No schema version declared in project.toml, defaulting to schema version 0.1\n")
		})

		it("should warn when unsupported keys, on tables the project owns, are declared with schema v0.1", func() {
			projectToml := `
[project]
authors = ["foo", "bar"]

# try to use buildpack.io table with version 0.1 - warning message expected
[[io.buildpacks.build.env]]
name = "JAVA_OPTS"
value = "-Xmx1g"

# something else defined by end-users - no warning message expected
[io.docker]
file = "./Dockerfile"

# some metadata - no warning message expected
[metadata]
foo = "bar"
`
			tmpProjectToml, err := createTmpProjectTomlFile(projectToml)
			if err != nil {
				t.Fatal(err)
			}

			_, err = ReadProjectDescriptor(tmpProjectToml.Name(), logger)
			h.AssertNil(t, err)
			h.AssertContains(
				t,
				readStdout(),
				"Warning: The following keys declared in project.toml are not supported in schema version 0.1:\n"+
					"Warning: - io.buildpacks.build.env\n"+
					"Warning: - io.buildpacks.build.env.name\n"+
					"Warning: - io.buildpacks.build.env.value\n"+
					"Warning: The above keys will be ignored. If this is not intentional, try updating your schema version.\n",
			)
		})

		it("should warn when unsupported keys, on tables the project owns, are declared with schema v0.2", func() {
			projectToml := `
[_]
schema-version = "0.2"
id = "foo"
version = "bar"
# typo in a key under valid table - warning message expected
versions = "0.1"

[[_.licenses]]
type = "foo"
# invalid key under a valid table - warning message expected
foo = "bar"

# try to use an invalid key under io.buildpacks - warning message expected
[[io.buildpacks.build.foo]]
name = "something"

# something else defined by end-users - no warning message expected
[io.docker]
file = "./Dockerfile"

# some metadata defined the end-user - no warning message expected
[_.metadata]
foo = "bar"

# more metadata defined the end-user - no warning message expected
[_.metadata.fizz]
buzz = ["a", "b", "c"]
`
			tmpProjectToml, err := createTmpProjectTomlFile(projectToml)
			if err != nil {
				t.Fatal(err)
			}

			_, err = ReadProjectDescriptor(tmpProjectToml.Name(), logger)
			h.AssertNil(t, err)

			// Assert we only warn
			h.AssertContains(
				t,
				readStdout(),
				"Warning: The following keys declared in project.toml are not supported in schema version 0.2:\n"+
					"Warning: - _.versions\n"+
					"Warning: - _.licenses.foo\n"+
					"Warning: - io.buildpacks.build.foo\n"+
					"Warning: - io.buildpacks.build.foo.name\n"+
					"Warning: The above keys will be ignored. If this is not intentional, try updating your schema version.\n",
			)
		})
	})
}

func createTmpProjectTomlFile(projectToml string) (*os.File, error) {
	tmpProjectToml, err := os.CreateTemp(os.TempDir(), "project-")
	if err != nil {
		log.Fatal("Failed to create temporary project toml file", err)
	}

	if _, err := tmpProjectToml.Write([]byte(projectToml)); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}
	return tmpProjectToml, err
}
