# Skaffold logs improvements

* Author(s): Prashant Arya (@prary)
* Design Shepherd: 
* Date: 7/05/2019
* Status: 

## Background

Kaniko pod exiting before errors could be read by Skaffold. Do a normal skaffold run and put a invalid 
base image or any other condition that would immediately(idea is to kill the kaniko pod before logger
is attached ) fail to build a image and exit kaniko pod. In this condition user would not get acutual 
reason of failure, he will just see following error

build step: building [someBaseImageUrl]: kaniko build for [someBaseImageUrl]: waiting for pod to complete: pod already in terminal phase: Failed   


Where as user should get actual error

     Error while retrieving image from cache: could not parse reference
     INFO[0002] Downloading base image gcr.io/invalidImageNameORUnReachableImage
     error building image: could not parse reference


This problem can also be exacted to a situation where user has put a such command which produce tons of output and before skaffold logger can pull all the logs kaniko pod  dies.

## Design

There should be a way where user can configure kaniko pod to stay alive for certain amount of time, so 
that no logs are flushed before being fetched. There should be podGracePeriodSecond which ensures that
kaniko pod stays alive for certain seconds after it finishes off building and pushing to registry. 

Possible solution to counter above situation would be create a extra container in same kaniko pod
which should continously monitor kaniko process. As soon as kaniko process terminate it should put 
other container to sleep for specified mount of time. Extra container could be called a side car 
which would not be very resource agnostic.

### Kaniko Pod Definition changes

```yaml
Containers: []v1.Container{
  {
    Name:            constants.DefaultKanikoContainerName,
    Image:           cfg.Image,
    Args:            args,
    ImagePullPolicy: v1.PullIfNotPresent,
    Env:             []v1.EnvVar{},
    VolumeMounts:    []v1.VolumeMount{},
  },
  {
    Name:            "side-car",
    Image:           constants.DefaultBusyboxImage,
    ImagePullPolicy: v1.PullIfNotPresent,
    Command: []string{"sh", "-c", "while [[ $(ps -ef | grep kaniko | wc -l) -gt 1 ]] ; do sleep 1; done; sleep " + cfg.PodGracePeriodSeconds},
  },
}
```
### Config changes

Extra podGracePeriodSeconds field would be added, so that user can configure it.


### Open Issues/Question
#1978


## Implementation plan

1. Modify kaniko pod definition
2. Monitoring kaniko pod, adding sleep when kaniko pod terminates

## Glossary

- side_car: Extra container running simulateously on kaniko pod and monitoring kaniko.
