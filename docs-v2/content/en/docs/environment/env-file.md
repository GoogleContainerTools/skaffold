---
title: "Load environment variables from a file"
linkTitle: "Load ENV from a file"
weight: 50
aliases: [/docs/concepts/env_file]
---

In Skaffold, a `skaffold.env` file can be defined in the project root directory to specify environment variables that Skaffold will load into the process. This provides a more organized and manageable way of setting environment variables, rather than passing them as command line arguments.

The `skaffold.env` file should be in the format of `KEY=value` pairs, with one pair per line. Skaffold will automatically load these variables into the environment before running any commands.

Here is an example `skaffold.env` file:

```txt
ENV_VAR_1=value1
ENV_VAR_2=value2
```

{{< alert title="Note" >}}
Values set in a `skaffold.env` file will not overwrite existing environment variables in the process.
{{< /alert >}}

### Setting Skaffold Flags with Environment Variables

In addition to loading environment variables from the `skaffold.env` file, Skaffold also allows users to set flags using environment variables. To set a flag using an environment variable, use the `SKAFFOLD_` prefix and convert the flag name to uppercase.

For example, to set the `--cache-artifacts` flag to `true`, the equivalent environment variable would be `SKAFFOLD_CACHE_ARTIFACTS=true`.

Here is an example usage in the `skaffold.env` file:

```txt
SKAFFOLD_CACHE_ARTIFACTS=true
SKAFFOLD_NAMESPACE=mynamespace
```

{{< alert title="Note" >}}
If a flag is set both in the `skaffold.env` file and as a command line argument, the value specified in the command line argument will take precedence.
{{< /alert >}}
