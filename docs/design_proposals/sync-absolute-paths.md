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
      type: absolute
```
___

## Design

### Config Changes

Added new configuration field (Type) in `SyncRule`.
```yaml
// SyncRule specifies which local files to sync to remote folders.
type SyncRule struct {
  ...
  ...
  ...

  // Type specifies the path type
  // For example: `"absolute"`
  Type string `yaml:"type,omitempty"`
}
```
The `Type` can only be "absolute" or empty.

The change of config doesn't affect older skaffold configuration files.


### Open Issues

#2898 

## Implementation plan
___

1. Add new config key `Type` to `sync.manual` and test schema validation.
2. Change path watchers to able to watch absolute paths.
3. Add files inside the absolute path to the dependencies list of the artifact.

___


## Integration test plan

Please describe what new test cases you are going to consider.
