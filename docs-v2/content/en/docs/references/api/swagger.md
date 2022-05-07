---
title: "HTTP API"
linkTitle: "HTTP API"
weight: 40
---

This is a generated reference for the [Skaffold API]({{<relref "/docs/design/api">}}) HTTP layer.

We also generate the [reference doc for the gRPC layer]({{<relref "/docs/references/api/grpc">}}).


<div id="swagger-ui"></div>

<script src="/swagger/swagger-ui-bundle.js"></script>
<script src="/swagger/swagger-ui-standalone-preset.js"></script>
<script>
    const DisableTryItOutPlugin = function () {
        return {
            statePlugins: {
                spec: {
                    wrapSelectors: {
                        allowTryItOutFor: () => () => false
                    }
                }
            }
        }
    }

    window.onload = function () {
        // Begin Swagger UI call region
        const ui = SwaggerUIBundle({
            url: "/api/skaffold.swagger.json",
            dom_id: '#swagger-ui',
            deepLinking: true,
            presets: [
                SwaggerUIBundle.presets.apis,
                // SwaggerUIStandalonePreset
            ],
            plugins: [
                SwaggerUIBundle.plugins.DownloadUrl,
                DisableTryItOutPlugin
            ],
            // layout: "StandaloneLayout"
        })
        // End Swagger UI call region

        window.ui = ui
    }
</script>

