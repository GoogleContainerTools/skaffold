# Title

* Author(s): Prashant Arya
* Design Shepherd: Tejal Desai
* Date: 3rd April 2018
* Status: 

## Background

At present if you run skaffold in seperate lab environment which doesn't have direct internet access
and need proxy setting then skaffold can directly access the envrionment variable of the lab and can
access the artifactory or registry but on the other hand builder like kaniko which spins up a seperate 
pod fails to build and push the built image to registry as it doesn't have proxy environment set. 
Hence skaffold run fails.

To overcome this problem we can set proxy in environment variable of kaniko pod defination and get the
skaffold running. We can take proxy variable in kaniko config section.

 
Please provide a brief explanation for the following questions:

1. Why is this required? So that skaffold can run in lab environment where access to internet is via proxy
2. If this is a redesign, what are the drawbacks of the current implementation? I don't have idea about the other builder,
3. Is there any another workaround, and if so, what are its drawbacks? I can't think of drawback
4. Mention related issues, if there are any. NA

Here is an example snippet for a new feature:

___
Setting proxy variable in pod definition 
```yaml
Containers: []v1.Container{
                {
                    Name:            constants.DefaultKanikoContainerName,
                    Image:           image,
                    Args:            args,
                    ImagePullPolicy: v1.PullIfNotPresent,
                    Env: []v1.EnvVar{
                        {
                        Name:  "GOOGLE_APPLICATION_CREDENTIALS",
                        Value: "/secret/kaniko-secret",
                        }
                        {
                        Name:  "http_proxy",
                        Value: "somevalue",
                        }
                        {
                        Name:  "https_proxy",
                        Value: "somevalue",
                        }
                    },
                },
```
Setting the proxy would give kaniko pod internet access. Where it can contact gcr or linux update server(any install command).
___

## Design

Please describe your solution. Please list any:

* new config changes
* interface changes
* design assumptions

For a new config change, please mention:

* Is it backwards compatible? If not, what is the deprecation policy? No idea 
  
```yaml
// ClusterDetails *beta* describes how to do an on-cluster build.
type ClusterDetails struct {
    
    // http_proxy     
    http_proxy string `yaml:"http_proxy,omitempty"`

    // https_proxy
    https_proxy string `yaml:"http_proxys,omitempty"`
    // PullSecret is the path to the secret key file.
    PullSecret string `yaml:"pullSecret,omitempty"`

    // PullSecretName is the name of the Kubernetes secret for pulling the files
    // from the build context and pushing the final image.
    // Defaults to `kaniko-secret`.
    PullSecretName string `yaml:"pullSecretName,omitempty"`

    // Namespace is the Kubernetes namespace.
    // Defaults to current namespace in Kubernetes configuration.
    Namespace string `yaml:"namespace,omitempty"`

    // Timeout is the amount of time (in seconds) that this build is allowed to run.
    // Defaults to 20 minutes (`20m`).
    Timeout string `yaml:"timeout,omitempty"`

    // DockerConfig describes how to mount the local Docker configuration into a pod.
    DockerConfig *DockerConfig `yaml:"dockerConfig,omitempty"`

    // Resources define the resource requirements for the kaniko pod.
    Resources *ResourceRequirements `yaml:"resources,omitempty"`
}

```

### Open Issues/Question
#2163
Please list any open questions here in the format.

**\<Question\>**
Do we need to set proxy for other builders as well.

## Implementation plan
As a team, we've noticed that larger PRs can go unreviewed for long periods of
time. Small incremental changes get reviewed faster and are also easier for
reviewers.

For a design feature, list a summary of tasks breakdown for e.g.:
For the example artifact sync proposal, some of the smaller tasks could be:
___

1. Add new field to cluster struct
2. Add logic to put all the environment variable in collection
3. Pass the collection to kaniko pod definition 
___
