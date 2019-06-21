# Skaffold logs improvements

* Author(s): Prashant Arya (@prary)
* Design Shepherd: Tejal Desai
* Date: 7/05/2019
* Status: 

#Disclaimer

This design doc covers only pod failure case

## Background

Consider a scenerio where kaniko pod fails due to kubernetes constraint, presently user gets pod already in terminal state only. User don't get any other info apart 
from pod why pod failed. In such scenerio skaffold should fetch all the event related to kaniko pod.   

Scenerio: User has put his custom init image and that image is not available for any reason 
than kaniko simply waits for init container to complete where init container fails to pull
the base image eventually kaniko also fails. In these kind of scenerios event logs related 
to kaniko pod would be extremely helpful where k8s clearly says ImagePullBackError.       

## Design

Possible solution to above problem statement would be fetch events log from k8s system.

### Kaniko Pod Definition changes

```yaml
  case v1.PodFailed or v1.PodUnknown or v1.PodPending::
   kubectl get events --namespace namespace-name --field-selector involvedObject.name=kaniko-pod-name
```

### Open Issues/Question
#1978
Should we pull event logs in case of all kind of pod failure? Yes we should pull logs for 
all failures in WaitForPodScheduled(), WaitForPodComplete() and WaitForPodInitialized().


## Implementation plan

1. Pulling events from  k8s in pkg/skaffold/kubernetes/wait.go WaitForPodComplete function 
