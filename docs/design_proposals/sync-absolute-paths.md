# Sync support absolute paths

* Author(s): Mustapha Yamie (@mustaphagit)
* Design Shepherd: 
* Date: 22/06/2021
* Status: [In review]
## Background

Currently skaffold does not support **absolute paths** in `sync` section. For example, if we want to sync the file in `/home/config/**` to container's `/config` folder we can't do that. Currently skaffold only supports relative paths.

Here is an example snippet for a new feature:
___
```yaml
sync:
  manual:
    - src: '/home/config/**'
      dest: /config/
```
___

## Design


### Open Issues

#2898 

## Implementation plan
___
1. Change path watchers to able to watch absolute paths.
2. Add files inside the absolute path to the dependencies list of the artifact.
3. Remove relative path restrictions.
___


## Integration test plan

Please describe what new test cases you are going to consider.
