# Title

* Author(s): Tejal Desai
* Design Shepherd: Brian de Alwis
* Date: 07/11/2021
* Status: Proposed

## Background

Currently, we get feedback on skaffold via skaffold HaTs survey. 
However, in this survey we don't get feedback on new or existing features. 
We also can't get enough information from our existing customers on how they are using certain features in case we want to make any changes.
e.g.
With render v2, we are changing how deploy works. This may affect our existing helm deployer users. 
See [#6166](https://github.com/GoogleContainerTools/skaffold/issues/6166)
## Design

### User survey Definition
In order to make sure only relevant surveys are shown to user, we have added a survey config.
```
type config struct {
	id           string
	promptText   string
	expiresAt    time.Time
	isRelevantFn func([]util.VersionedConfig) bool
	link         string
}

```
The survey config has two key fields
1) expiresAt - This decided the lifetime of the survey. e.g. if we are collecting user feedback to get feedback on helm deployer re-design, then it makes sense to target the survey towards helm deployer user for a limited time until the re-design requirement phase is complete.
2) isRelevantFn - This function determines if the user is a target audience for the survey. For the above example, it only makes sense to show the survey for helm deployer users, like this.
```
{
  id: helmID,
  expiresAt: time.Date(2021, time.August, 14, 00, 00, 00, 0, time.UTC),
  isRelevantFn: func(cfgs []util.VersionedConfig) bool {
	for _, cfg := range cfgs {
		v1Cfg := cfg.(*latestV1.SkaffoldConfig)
		if h := v1Cfg.Deploy.HelmDeploy; h != nil {
			return true
	        }
        }
	return false
 },
link: helmURL,
},

```
something similar can be imaged for multi-module users
```
  isRelevantFn: func(cfgs []util.VersionedConfig) bool {
	return len(cfgs) > 1
 },
```

### How to prompt users to take user surveys

#### Rules to show survey prompt
When prompting users with surveys, we need to keep in mind 2 important rules
1) Don't prompt users too often.
   Currently, we only prompt users to fill in HaTs survey every two weeks until they fill.
2) Don't prompt users if they have already taken the survey.
   Currently, we only prompt users to fill in Hats Survey if they haven't taken it in last 3 months.

For non HaTs surveys or user surveys, we will follow the same rules,
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
2) If Hats survey was not taken in last 3 months (stored in  `SurveyConfig.LastTaken`) and
3) If survey prompt was not shown `SurveyConfig.LastPrompted` in last 2 weeks.

This behavior will be changed to `ShouldDisplaySurveyPrompt` returns true if
1) If survey prompt is not disabled and
2)  If survey prompt was not shown `SurveyConfig.LastPrompted` in last 2 weeks and
3) If there is an active relevant user survey is available or Hats survey was not taken in last 3 months

If both active relevant user survey is available or Hats survey was not taken, then skaffold will give preference to user survey over hats survey.
Since, hats survey never expire,
- if user takes the user survey this time, hats survey will be prompted the next time.
-  if the user never takes the user survey, the hats survey will be prompted once the user survey expires.
   **Note User surveys should not run longer than a quarter so that we don't reduce the volume of Hats Survey,
```
pkg survey/config

def init() {
  for _, s := range surveys {
     if s.id != hats.id {
       if s.expiresAt.IsZero() || s.expiresAt.Sub(time.Now) > 90days
          panic(fmt.Errorf("survey %s is running longer than 90 days. This will starve users from being shown the hats survey."))
    }
  }
}

```

Alternate options:
1) Another option is to show the Hats survey right after user survey. 
   Look into if we can keep the same Hats response Form.

## Metrics
1) Measure sample responses to user surveys
2) Monitor the number of Hats responses per month is not reducing.

## Alternate Methods
Other methods to get feedback is manually via pinging slack users to fill in a survey.
However, this requires a core member to manually remind users from time to time to fill in a survey.


## Implementation plan
- [ ] add survey config struct
- [ ] add `-id` flag to survey command with default as `hats`
- [ ] add `UserSurvey` struct to skaffold global config
- [ ] change prompt logic to show active relevant prompts
- [ ] Change set and unset command to set user survey fields.
