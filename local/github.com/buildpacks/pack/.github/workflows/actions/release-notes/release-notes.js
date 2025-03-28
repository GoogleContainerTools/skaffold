const {promises: fs} = require('fs');
const YAML = require('yaml');

module.exports = async (github, repository, milestone, configPath) => {
  return await fs.readFile(configPath, "utf-8")
    .then(content => YAML.parse(content))
    .then(config => {
      let labelGroups = config.labels
      let weightSortFunc = (a, b) => labelGroups[a].weight - labelGroups[b].weight

      console.log("looking up PRs for milestone", milestone, "in repo", repository);
      return github.paginate("GET /search/issues", {
        q: `repo:${repository} is:pr is:merged milestone:${milestone}`,
      }).then(issues => {
        console.log("Issues count:", issues.length);

        let groupedIssues = groupIssuesByLabels(issues, Object.keys(labelGroups));

        // generate issues list
        let output = "";
        for (let key of Object.keys(labelGroups).sort(weightSortFunc)) {
          let displayGroup = labelGroups[key]
          let issues = (groupedIssues[key] || []);
          console.log(key, "issues:", issues.length);

          if (issues.length > 0) {
            output += `### ${displayGroup.title}\n\n`;
            if (displayGroup.description) {
              output += `${displayGroup.description.trim()}\n\n`;
            }
            issues.forEach(issue => {
              output += createIssueEntry(issue);
            });
            output += "\n";
          }
        }

        let hiddenIssues = groupedIssues[""] || [];
        console.warn("Issues not displayed: ", hiddenIssues.length);
        if (hiddenIssues.length > 0) {
          console.warn(" - " + hiddenIssues.map(issue => issue.number).join(", "));
        }

        // generate contributors list
        if (
          config
          && config.sections
          && config.sections.contributors
          && config.sections.contributors.title
        ) {
          output += `## ${config.sections.contributors.title}\n\n`;

          if (config.sections.contributors.description) {
            output += `${config.sections.contributors.description.trim()}\n\n`;
          }

          let uniqueFunc = (value, index, self) => self.indexOf(value) === index
          output += issues
            .map(issueContrib)
            .filter(uniqueFunc)
            .sort()
            .map(v => `@${v}`)
            .join(", ")
        }

        return output.trim();
      });
    })
};

function createIssueEntry(issue) {
  return `* ${issue.title} (#${issue.number} by @${issueContrib(issue)})\n`;
}

function issueContrib(issue) {
  return issue.user.login;
}

function groupIssuesByLabels(issues, labels) {
  return issues.reduce((groupedMap, issue) => {
    let typeLabel = issue.labels
      .filter(label => labels.includes(label.name))
      .map(label => label.name)[0] || "";

    (groupedMap[typeLabel] = groupedMap[typeLabel] || []).push(issue);

    return groupedMap;
  }, {})
}