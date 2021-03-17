# New command `skaffold inspect`

* Author(s): gsquared94
* Date: 2021-03-12
* Status: [Under review]

## Background

Prior to the implementation of multiple configuration support in `skaffold` it was easy to take a look at the single `skaffold.yaml` file and understand the state of things. Now, a single `skaffold.yaml` configuration file can import other local and remote configurations as dependencies and we can have a complicated graph where it's not easy to predetermine what are all the artifacts that are going to be built and deployed by specifying a `module` filter when running skaffold (like `skaffold dev -m foo -m bar`).

The Cloud Code IDEs (VSCode and IntelliJ) integrate with Skaffold and directly read and parse the `skaffold.yaml` file for information like the list of profiles available and the list of all artifacts.

We should consolidate all such querying of the skaffold configuration behavior into a single command:
```
  skaffold inspect [flags] [options]
```
___

## Design
```
skaffold inspect --help
Display all resolved skaffold configurations.

 You can use --output jsonpath={...} to extract specific values using a jsonpath expression.

Examples:
  # Show all resolved configurations with applied profiles.
  skaffold inspect
  
  # Show all resolved configurations with profiles not applied.
  skaffold inspect --raw
  
  # Get all named configs (modules) as a list
  skaffold inspect -o jsonpath='{.metadata[?(@.name != "")].name}'

  # Get all profile names.
  skaffold inspect -o jsonpath='{.profiles[*].name}'

Options:
      --allow-missing-template-keys=true: If true, ignore any errors in templates when a field or map key is missing in
the template. Only applies to jsonpath output formats.
  -o, --output='yaml': Output format. One of:
json|yaml|jsonpath.
      --raw=false: Raw output will not apply profiles and just display the contents of the parsed files.

Usage:
  skaffold inspect [flags] [options]

Use "skaffold options" for a list of global command-line options (applies to all
commands).
```

## Implementation plan

1. Add command with flag defaults and hidden.
2. Implement `jsonpath` filtering of output.
3. Add integration tests.
4. Unhide flag.
___

## Integration test plan

Test command with a combination of flags against the [`multi-config-microservices`](examples/multi-config-microservices) example.
