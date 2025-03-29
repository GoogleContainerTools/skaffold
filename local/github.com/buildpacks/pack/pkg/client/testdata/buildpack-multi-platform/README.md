When creating multi-platform buildpacks, the root buildpack.toml file must be copied into each
plaform root folder; this operation must be done by the caller of the method:

`PackageBuildpack(ctx context.Context, opts PackageBuildpackOptions) error`

To simplify the tests, the buildpack.toml is already copied in each buildpack folder.
