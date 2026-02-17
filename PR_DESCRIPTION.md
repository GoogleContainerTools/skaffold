## PR Description: Recovering Multi-Arch Support and Compatibility

### Problem
The standard `skaffold` library had compatibility issues when used within `octopilot-pipeline-tools` (`op`):
1.  **Go Version Mismatch**: `op` requires Go 1.25.6, but upstream `skaffold` moved ahead, preventing `op` from importing it as a library.
2.  **Multi-Arch Build Failures**: When using Buildpacks, `skaffold` relies on the Docker daemon. This fails for multi-arch builds (e.g., `linux/amd64,linux/arm64`) because the local Docker daemon cannot easily hold/export manifest lists for multiple architectures simultaneously without special handling.
3.  **Lifecycle Compatibility**: Older lifecycle versions were causing issues with newer buildpacks.

### Changes
-   **Downgrade Go**: Downgraded Go version to `1.25.6` to match `op`'s requirements.
-   **Update Lifecycle**: Updated `github.com/buildpacks/lifecycle` to `v0.21.0`.
-   **Pack Integration**: Updated dependencies to use `octopilot/pack` fork which contains critical fixes for registry interactions.
-   **Publish Fix**: Enabled `Publish: true` and ensured the requested tag is used when pushing buildpacks artifacts. This allows `op` (via its direct pack integration path) to bypass the Docker daemon export and push directly to the registry, enabling valid multi-arch manifests.

### Verification
Verified end-to-end using `op` on the `sample-static-rust-axum` repository:
```bash
op build --repo ttl.sh/test-multi-arch --push --platform linux/amd64,linux/arm64
```
**Result**: Successfully built and pushed multi-arch images for both Node.js frontend and Rust API.
