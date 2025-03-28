//go:build acceptance

package invoke

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	h "github.com/buildpacks/pack/testhelpers"
)

type PackFixtureManager struct {
	testObject *testing.T
	assert     h.AssertionManager
	locations  []string
}

func (m PackFixtureManager) FixtureLocation(name string) string {
	m.testObject.Helper()

	for _, dir := range m.locations {
		fixtureLocation := filepath.Join(dir, name)
		_, err := os.Stat(fixtureLocation)
		if !os.IsNotExist(err) {
			return fixtureLocation
		}
	}

	m.testObject.Fatalf("fixture %s does not exist in %v", name, m.locations)

	return ""
}

func (m PackFixtureManager) VersionedFixtureOrFallbackLocation(pattern, version, fallback string) string {
	m.testObject.Helper()

	versionedName := fmt.Sprintf(pattern, version)

	for _, dir := range m.locations {
		fixtureLocation := filepath.Join(dir, versionedName)
		_, err := os.Stat(fixtureLocation)
		if !os.IsNotExist(err) {
			return fixtureLocation
		}
	}

	return m.FixtureLocation(fallback)
}

func (m PackFixtureManager) TemplateFixture(templateName string, templateData map[string]interface{}) string {
	m.testObject.Helper()

	outputTemplate, err := os.ReadFile(m.FixtureLocation(templateName))
	m.assert.Nil(err)

	return m.fillTemplate(outputTemplate, templateData)
}

func (m PackFixtureManager) TemplateVersionedFixture(
	versionedPattern, version, fallback string,
	templateData map[string]interface{},
) string {
	m.testObject.Helper()
	outputTemplate, err := os.ReadFile(m.VersionedFixtureOrFallbackLocation(versionedPattern, version, fallback))
	m.assert.Nil(err)

	return m.fillTemplate(outputTemplate, templateData)
}

func (m PackFixtureManager) TemplateFixtureToFile(name string, destination *os.File, data map[string]interface{}) {
	_, err := io.WriteString(destination, m.TemplateFixture(name, data))
	m.assert.Nil(err)
}

func (m PackFixtureManager) fillTemplate(templateContents []byte, data map[string]interface{}) string {
	tpl, err := template.New("").
		Funcs(template.FuncMap{
			"StringsJoin": strings.Join,
			"StringsDoubleQuote": func(s []string) []string {
				result := []string{}
				for _, str := range s {
					result = append(result, fmt.Sprintf(`"%s"`, str))
				}
				return result
			},
			"StringsEscapeBackslash": func(s string) string {
				result := []rune{}
				for _, elem := range s {
					switch {
					case elem == '\\':
						result = append(result, '\\', '\\')
					default:
						result = append(result, elem)
					}
				}
				return string(result)
			},
		}).
		Parse(string(templateContents))
	m.assert.Nil(err)

	var templatedContent bytes.Buffer
	err = tpl.Execute(&templatedContent, data)
	m.assert.Nil(err)

	return templatedContent.String()
}
