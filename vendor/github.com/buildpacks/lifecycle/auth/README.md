# auth

## Skipping Vendor Specific Keychains

The auth package has configuration available to skip vendor specific keychain implementations. If you are a platform handling credentials yourself, you may want to skip loading these keychains. This can improve performance as the helpers automatically get invoked based on the hosting environment and the registries being interacted with.

Set any of the following to `true` to skip loading the vendor keychain.

`CNB_REGISTRY_AUTH_KEYCHAIN_SKIP_AMAZON` - set to skip Amazon/AWS's ECR credhelper.
`CNB_REGISTRY_AUTH_KEYCHAIN_SKIP_AZURE` - set to skip Microsoft/Azure's ACR credhelper
