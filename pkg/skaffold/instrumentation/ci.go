package instrumentation

var ciMap = map[string]string{
	"TF_BUILD":           "azure-pipelines",
	"bamboo_buildKey":    "bamboo",
	"BUILDKITE":          "buildkite",
	"CIRCLECI":           "circle-ci",
	"CIRRUS_CI":          "cirrus-ci",
	"CODEBUILD_BUILD_ID": "code-build",
	"GITHUB_ACTIONS":     "github-actions",
	"GITLAB_CI":          "gitlab-ci",
	"HEROKU_TEST_RUN_ID": "heroku-ci",
	"JENKINS_URL":        "jenkins",
	"TEAMCITY_VERSION":   "team-city",
	"TRAVIS":             "travis-ci",
}
