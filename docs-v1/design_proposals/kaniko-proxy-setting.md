# Title

* Author(s): Prashant Arya
* Design Shepherd: Tejal Desai
* Date: 3rd April 2018
* Status:  Approved/

## Background

At present if you run skaffold in seperate lab environment without direct internet access, skaffold command with kaniko builds fail. This is because, kaniko spins up a seperate pod which do not have the proxy information set. 

To overcome this problem we can add proxy environment setting to kaniko config section to plumb it through to kaniko pod config.
 
Here is an example of new kaniko Pod config will look like with http proxy information.

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
We will be adding 2 new config variables in `ClusterDetails` config section.
For a new config change, please mention:

  
```yaml
// ClusterDetails *beta* describes how to do an on-cluster build.
type ClusterDetails struct {
    
    // HTTP_PROXY sets the "http_proxy" environment variable to the pod running cluster build.      
    HTTP_PROXY string `yaml:"httpProxy,omitempty"`

    // HTTPS_PROXY sets the "https_proxy" environment variable to the pod running cluster build. 
    HTTPS_PROXY string `yaml:"httpsProxy,omitempty"`
    
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


**\<Question\>**

Do we need to set proxy for other builders as well?

Resolution: No. As of now we don't have anyother cluster builder.

## Implementation plan
___

1. Add new field to cluster struct
2. Add logic to put all the environment variable in collection
3. Pass the collection to kaniko pod definition 
___
