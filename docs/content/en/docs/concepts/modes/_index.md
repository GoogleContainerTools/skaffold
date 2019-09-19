---
title: "Operating modes"
linkTitle: "Operating modes"
weight: 30
---

This page discusses various operating modes of Skaffold.


Skaffold provides two separate operating modes:

* `skaffold dev`, the continuous development mode, enables monitoring of the
    source repository, so that every time you make changes to the source code,
    Skaffold will build and deploy your application.
* `skaffold run`, the standard mode, instructs Skaffold to build and deploy
    your application exactly once. When you make changes to the source code,
    you will have to call `skaffold run` again to build and deploy your
    application.

Skaffold command-line interface also provides other functionalities that may
be helpful to your project. For more information, see [CLI References](/docs/references/cli).

