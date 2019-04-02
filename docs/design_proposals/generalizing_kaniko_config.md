# Generalizing Container Registry config for Cluster builds

* Author(s): venkatesh
* Design Shepherd: 
* Date: 
* Status: 

## Background

Kaniko supports 2 types of registries GCR and Docker, with former being mandatory always. 
The implementation itself is repository specific and not extensible as we need to keep adding more logic for every new type of repository and keep maintaining compatibilty with registries/credhelpers.

The current looks like below( I have kept only required pieces from original code snippet):
``` go
type ClusterDetails struct {
       // GCR creds
	PullSecret     string        `yaml:"pullSecret,omitempty"`
	PullSecretName string        `yaml:"pullSecretName,omitempty"`

	DockerConfig   *DockerConfig `yaml:"dockerConfig,omitempty"`
}

// DockerConfig contains information about the docker `config.json` to mount.
type DockerConfig struct {
	Path       string `yaml:"path,omitempty"`
	SecretName string `yaml:"secretName,omitempty"`
}
```
### Breaking down the requirements and generalizing:

If you observe the above piece of configuration, you will see both creds have same 2 fields, SecretName/SecretPath. 
When we look into how they behave differently, we can observe the differences between them as follows:
1. The path where to mount the config varies based on secret
1. Some of them(like GCR) might require env variables to be set.

If we can generalize the above points, then any configuration(GCR/ER/private registry/S3/GCS, etc..) breaks down to following steps:
1. Create a secret with the configuration
2. Mount the config in specific location in pod
3. In some cases set some env variables, for the same.


## Design

The same above can be re-configured in a more generic way as follows:

``` go
type ClusterDetails struct {
	SecretConfigs []*SecretConfig   `yaml:"secrets,omitempty"`
	env           map[string]string `yaml:"env,omitempty"`
}

// DockerConfig contains information about the docker `config.json` to mount.
type SecretConfig struct {
	LocalPath  string `yaml:"localPath"`
	SecretName string `yaml:"secretName,omitempty"`
	MountPath  string `yaml:"mountPath,omitempty"`
}
```
### How different configurations fit into this scheme:
#### GCR/GCS
current scheme:
``` yaml
build:
  cluster:
    pullSecretName: e2esecret
```
new scheme:
``` yaml
build:
  cluster:
    secrets: 
    - secretName: e2esecret
      mountPath:  "/secret"
    env:
       "GOOGLE_APPLICATION_CREDENTIALS" : "/secret/kaniko-secret"
```
#### Private Registry:
current scheme:
``` yaml
build:
  cluster:
    dockerConfig:
      secretName: e2esecret
```
new scheme:
``` yaml
build:
  cluster:
    secrets: 
    - secretName: docker-secret
      mountPath:  "/kaniko/.docker"
```
#### ECR:
current scheme: None
new scheme( [pushing-to-amazon-ecr](https://github.com/GoogleContainerTools/kaniko/#pushing-to-amazon-ecr)) :
``` yaml
build:
  cluster:
    secrets: 
    - secretName: aws-secret
      mountPath:  "/root/.aws/"
    - secretName: docker-config
      mountPath:  "/kaniko/.docker/"
```
### Pros :
1. more extensible and future proof.
2. Can support varies use cases, like setting http_proxy, source from github, etc. to name a few
3. Not limited to kaniko, can be extended to any cluster builder, keeping in spirit of recent changes.

### Cons:
1. Slightly more config for users, specifically for GCR

### Open Issues/Question

Please list any open questions here in the format.

**\<Question\>**

Resolution: Please list the resolution if resolved during the design process or
specify __Not Yet Resolved__

## Implementation plan
1. We will implement the new configurations first, so ECR can be supported
2. Write skaffold fix for converting GCR/Docker secrets and give dprecation warnings for same
3. Remove GCR/Docker specific config and code for configuring secrets

___


## Integration test plan

1. Add unit tests to verify pod is getting configured correctly
2. Add new integration tests for all three types of registries
3. Not sure if its possible to add ECR/docker secrets