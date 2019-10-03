# Title

* Author(s): Tejal Desai
* Design Shepherd: balintp
* Date: 10/03/2019
* Status: Draft

# Background

This design document is to mitigate the issues identified [here](https://github.com/GoogleContainerTools/skaffold/issues/1930).

Current Implementation

Skaffold, currently gets DefaultRepo from

1. Use default Repo from --default-repo command line flag
   - E.g.: `skaffold build --default-repo=gcr.io/project`
1. Check for Default Repo specified in kubeContext
   - E.g `default-repo: gcr.io/project`
1. Finally, get default repo from skaffold Global Config.
    ```
    cat ~/skaffold/config
    default-repo: gcr.io/project`
    ```
If the default repo is specified, then following substitution logic is performed.


| Scenarios | Original Image | Default Repo | Resulted Image Value |
| --- | --- | --- | --- |
| **Scenario 1** When a GCR image belongs to default project, then it is returned as is. | gcr.io/**defaultPrj**/name| gcr.io/**defaultPrj**| gcr.io/**defaultPrj**/name|
| **Scenario 2** When a GCR image contains a prefix that matches GCR Default Repository | gcr.io/**defaultPrj**/path1/name | gcr.io/**defaultPrj**/**path** | gcr.io/**defaultPrj**/**path**/path1/name |
| **Scenario 3** When a GCR image does not overlap with the GCR Default Repository, we concatenate both the strings. | gcr.io/**prodPrj/path1/name** | gcr.io/**defaultPrj** & <br /> gcr.io/**defaultPrj/path2**| gcr.io/**defaultPrj/gcr.io/projPrj/path1/name**  & <br />gcr.io/**defaultPrj/path2/gcr.io/projPrj/path1/name** <br /> Shd be: <br /> **gcr.io/defaultPrj/path1/name** & <br />**gcr.io/defaultPrj/path2/path1/name**  |
| **Scenario 4** When a GCR image is pushed to non GCR repository, we replace all "/" to "_" and concatenate | gcr.io/**myPrj/image** | aws_myAccountId.amazonaws.com| aws_myAccountId.amazonaws.com/**gcr_io_myPrj_image** <br /> **Shd be:** <br />aws_myAccountId.amazonaws.com/image |
| **Scenario 5** When a non GCR image is pushed to non GCR repository, we replace all "/" to "\_" and concatenate. | aws_prodId_dkr.ecr.region.amazonaws.com/**image**| aws_devId_dkr.ecr.region.amazonaws.com & <br />aws_devId_dkr.ecr.region.amazonaws.com/ns | aws_devId_dkr.ecr.region.amazonaws.com/aws_prodId_dkr.ecr.region.amazonaws.com_image & <br /> aws_devId_dkr.ecr.region.amazonaws.com/ns/aws_prodId_dkr.ecr.region.amazonaws.com_image <br/> **Shd be:** <br /> aws_devId_dkr.ecr.region.amazonaws.com/image & <br />aws_devId_dkr.ecr.region.amazonaws.com/ns/image |
| **Scenario 6** When an image is pushed to docker repository at 127.0.0.1:5000 | image |127.0.0.1:5000/ | 127.0.0.1:5000/**image** |
| **Scenario 7** When a docker image from 1 repository is pushed to docker repository at 127.0.0.1:5000 | registry.hub.docker.com/library/skaffold-example| 127.0.0.1:5000/test/ & <br /> 127.0.0.1:5000/test/path| 127.0.0.1:5000/test/registry\_hub\_docker\_com\_library\_skaffold-example & <br/> 127.0.0.1:5000/test/path/registry\_hub\_docker\_com\_library\_skaffold-example <br/> **Shd be:** <br/> 127.0.0.1:5000/test/library/skaffold\_example &  <br/>  127.0.0.1:5000/test/path/library/skaffold\_example |
| **Scenario 8** We don't know the structure of the registry | artifactory:5000/test/path/library/skaffold_example | gcr.io/myproj | ??? |
| **Scenario 9** When a GCR image is pushed to general register | **gcr.io/balintp-gcp-lab/foo/bar** | artifactory:5000/path & <br/> artifactory:5000/test/path | artifactory:5000/path/foo/bar & <br/>artifactory:5000/test/path/foo/bar|

The images produced are all correct! However the behaviour is surprising to users.

Especially Scenario 3 and after.

We could definitely do this better by creating Specific Registry Implementation and registry specific replace rules.

## Proposed Solution:
- We assume a structure on registries
  - **GCR**: `project{/path1/path2/...}`
  
    ```
    type GCRegistrystruct {
      Project string
      Paths[]string
    }
    ```
  - **AWS**: `repo{/ns}`
    ```
    type ECRegistrystruct {
      Domain string
      Namespace string
    }
    ```
  - **DockerHub**: `registry{/path1/path2/...}`
    ```
    type DockerHubstruct {
      RepositoryUrl string
      UserName string
    }
    ```
  - **GenericRegistry**
    ```
    type GenericRegistrystruct {
     Registry string
    }
    ```
- Introduce an interface Registry and  ImageRegistry which implements the following interface
  ```
  type Registry interface {
    // Name returns the string representation of the registry
    Name() string
    // Replace replaces the current registry in a given image name to input registry
    Update(reg Registry) Registry
    // Prefix gives the prefix for replacing the registry
    Prefix() string
    // Postfix gives the postfix for replacing the registry
    Prefix() string
  }
  
  type Image string{
    // Registry returns the registry for a given image.
    Registry() Registry
    // Name returns the image name
    Name() string
    // Replace updates the Registry for the image to registry new Registry reg and return the updated Image.
    Update(reg Registry) string
  }
  ```
The proposed Solution will implement Registry Specific Image interfaces, `GCRImage` and `ECRImage`.

Similarly, there will be specific implementation for `GCRegistry` (Google Container Registry) and `ECRegistry` (Elastic Container Registry).


### Google Container Registry Image Handling

GCRegistry **gcr.io/k8skaffold/subproject** , the struct would look like this.
  ```
  type GCRegistrystruct {
     Project string
     Paths[] string
  }
  ```
  Where `Project=k8skaffold` and `Paths=[]string{"subproject"}`
 
- For replacing a GCR registry in an image
  ```
  GCRImage("gcr.io/prod/subproject/img").
    Update(GCRegistry("gcr.io/dev/another/path1"))
  ```
  will return a GCR Image `gcr.io/dev/another/path1/subproject`.
  - The Project will be from the replaceTo GCR Registry
  - `Paths := append(replaceTo.Paths, original.Paths…)`
    <br/>Hence, the new GCRegistry will be:
    <br/>`Project: dev`
    <br/>`Paths: []string{"another", "path1", "subproject"}`

- For replacing a non GCR registry to a GCRegistry
  ```
  SomeImage("some-reg/image").
    Update(GCRegistry("gcr.io/dev/another/path")) 
  ```
  - The Project will be from the replaceTo GCR Registry
  - `Paths := append(replaceTo.paths, original.Name())`
    <br/>Hence, the new GCRegistry will be:
    <br/>`Project: dev`
    <br/>`Paths: []string{"another", "path1", "some-reg"}`

### Elastic Container Registry Image Handling
For ECRegistry `aws_prodId_dkr.ecr.region1.amazonaws.com/ns`, the struct would look like this.
```
type ECRegistry struct {
   Domain string
   Namespace string
}
```
Where `Domain=aws_prodId_dkr.ecr.region1.amazonaws.com` and `Namespace=test` 

- For replacing a ECR registry in an image
  ```
  ECRegistry("aws_devId_dkr.ecr.region2.amazonaws.com").
    Replace(ECRegistry("aws_devId_dkr.ecr.region2.amazonaws.com"))
  ``` 
  The Domain will be from the replaceTo ECRegistry<br/>
  Hence, the new ECRegistry will be: <br/>
  `Domain:aws_devId\dkr.ecr.region2.amazonaws.com`
  
  Let us consider a registry replace scenarios where registries are of different types.

- For GCR Image `gcr.io/k8skaffold/subproject/example` ,


### Generic Container Registry Image Handling

In case, we cannot detect the registry type, we will use the GenericContainerRegistry and GenericContainerRegitryImage implementation.
```
type GenericContainerRegistry struct {
  Name string
}
```
And
```
type GenericContainerRegistryImagestring{
  Registry GenericContainerRegistry
  Name string
}
```
The Replace rules for GenericContainerRegistry are as follows:
```
GenericContainerImage("some/name").
  Replace((GenericContainerRegistry("another"))
```
1. Create a new GenericContainerRegistry with Name = replaceToRegistry.Name()

The result will be :

GenericContainerRegistry{Name: another}

## Pros and Cons
### Pros:

1. This solution allows to support other Registries in future.
2. Can support multiple replace strategies.

## Cons:
1. More interfaces and implementations.