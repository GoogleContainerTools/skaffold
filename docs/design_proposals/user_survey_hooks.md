# Title

* Author(s): Tejal Desai
* Design Shepherd: Brian de Alwis
* Date: 07/11/2021
* Status: Proposed

## Background

Currently, we get feedback on Skaffold via Skaffold HaTS survey. 
However, in this survey we cannot prompt users for feedback on existing features or new proposed features. 
It is difficult for us to solicit feedback on existing features and the impact of proposed changes.
For example, with render v2, we are changing how deploy works. This may affect our existing helm deployer users. 
See [#6166](https://github.com/GoogleContainerTools/skaffold/issues/6166)

## Design

This document proposes to extend the existing HaTS survey framework to incorporate a set of user surveys.

### User survey definition
In order to make sure only relevant surveys are shown to user, we have added a survey config.
```
type config struct {
	id           string
	// promptText is shown to the user and should be formatted so each line should fit in < 80 characters.
	// For example: `As a Helm user, we are requesting your feedback on a proposed change to Skaffold's integration with Helm.`
	promptText   string
	// startsAt mentions the date after the users survey should be prompted. This will ensure, skaffold team can finalize the survey 
	// even after release date.
	startsAt time.Time
	// expiresAt places a time limit of the user survey. As users are only prompted every two weeks
	// by design, this time limit should be at least 4 weeks after the upcoming release date to account
	// for release propagation lag to Cloud SDK and Cloud Shell.
	expiresAt    time.Time
	isRelevantFn func([]util.VersionedConfig) bool
	URL         string
}

```
The survey config has two key fields
1) expiresAt - This decided the lifetime of the survey. e.g. if we are collecting user feedback to get feedback on helm deployer re-design, then it makes sense to target the survey towards helm deployer user for a limited time until the re-design requirement phase is complete.
2) isRelevantFn - This function determines if the user is a target audience for the survey. For the above example, it only makes sense to show the survey for helm deployer users, like this.
```
{
  id: helmID,
  expiresAt: time.Date(2021, time.August, 14, 00, 00, 00, 0, time.UTC),
  isRelevantFn: func(cfgs []util.VersionedConfig, command string) bool {
	for _, cfg := range cfgs {
		if v1Cfg, ok  := cfg.(*latestV1.SkaffoldConfig) ; ok {
		    if h := v1Cfg.Deploy.HelmDeploy; h != nil {
			    return true
	        }
	    }    
    }
	return false
 },
URL: helmURL,
},

```
For multi-module users, we could use something like the following:
```
  isRelevantFn: func(cfgs []util.VersionedConfig, _ string) bool {
	return len(cfgs) > 1
 },
```

### How to prompt users to take user surveys

#### Rules to show survey prompt
When prompting users with surveys, we need to keep in mind 2 important rules
1) *Don't prompt users too often.*
   Currently, we only prompt users to fill in HaTs survey every two weeks until they fill.
2) *Don't prompt users if they have already taken the survey.*
   Currently, we only prompt users to fill in HaTS Survey if they haven't taken it in last 3 months.

For non HaTS surveys or user surveys, we will follow the same rules,
1) Prompt users to fill in the survey once 2 weeks if its relevant to them until they fill it.
2) Stop prompting once they have taken the survey.

The user survey information will be tracked in the existing `Survey` config in the skaffold global config
```
// SurveyConfig is the survey config information
type SurveyConfig struct {
	DisablePrompt *bool           `yaml:"disable-prompt,omitempty"`
	LastTaken     string          `yaml:"last-taken,omitempty"`
	LastPrompted  string          `yaml:"last-prompted,omitempty"`
+	UserSurveys []*UseSurvey `yaml:"user-surveys,omitempty"`
}

type UserSurvey struct {
	ID              string `yaml:"id"`
	Taken        *bool  `yaml:"taken,omitempty"`
}
```


### Implementation details.
The current `ShouldDisplaySurveyPrompt` returns true only if
1) If survey prompt is not disabled and
2) If HaTS survey was not taken in last 3 months (stored in  `SurveyConfig.LastTaken`) and
3) If survey prompt was not shown `SurveyConfig.LastPrompted` in last 2 weeks.

This behavior will be changed to `ShouldDisplaySurveyPrompt` returns true if
1) If survey prompt is not disabled and
2)  If survey prompt was not shown `SurveyConfig.LastPrompted` in last 2 weeks and
3) If there is an active relevant user survey is available or HaTS survey was not taken in last 3 months

If both active relevant user survey is available or HaTS survey was not taken, then Skaffold will give preference to user survey over hats survey.
Since, HaTS survey never expire,
- if user takes the user survey this time, HaTS survey will be prompted the next time.
-  if the user never takes the user survey, the HaTS survey will be prompted once the user survey expires.
   **Note User surveys should not run longer than a quarter so that we don't reduce the volume of HaTS Survey,

## Alternate Methods
Other methods to get feedback is manually via pinging slack users to fill in a survey.
However, this requires a core member to manually remind users from time to time to fill in a survey.
Another disadvantage is users could form a biased opinion due to the questions asked in the user survey and 
rate low on the NPS score.


## Implementation plan
- [ ] add survey config struct
- [ ] add `-id` flag to survey command with default as `hats`
- [ ] add `UserSurvey` struct to skaffold global config
- [ ] change prompt logic to show active relevant prompts
- [ ] Change set and unset command to set user survey fields.
