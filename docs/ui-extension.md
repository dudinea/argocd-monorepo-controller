# UI Extension

The Monorepo Controller ships an [Argo CD UI extension](https://argo-cd.readthedocs.io/en/latest/developer-guide/extensions/ui-extensions/)
that adds a **Change Revision** item to the Application status panel, the bar at
the top of the application view where the sync status is shown.

!!! warning "Work in progress"
    The extension is currently a proof of concept: the panel item renders a
    fixed string. Displaying the actual [Change Revision](terminology.md#change-revision)
    of the application, and one entry per source for multi source applications,
    is the next step.

## How it works

The extension is a single JavaScript file, `ui/extension-monorepo-controller.js`.
Argo CD UI extensions are served by `argocd-server`: it collects every file named
`extension*.js` under `/tmp/extensions` and serves them concatenated as
`/extensions.js`, which the UI loads when the page is rendered. Each extension
registers itself through the `extensionsAPI` global variable.

The extension reads everything it needs from the Application object that the UI
already has, so it needs no backend component and no access to the Monorepo
Controller or its repo server.

## Installing for development

The `install-ui-extension-local` target stores the extension in a ConfigMap and
patches the `argocd-server` Deployment to mount it under `/tmp/extensions`:

```bash
make install-ui-extension-local
```

Use `ARGOCD_NAMESPACE` if Argo CD is not installed in the `argocd` namespace:

```bash
ARGOCD_NAMESPACE=my-argocd make install-ui-extension-local
```

To revert the change to the `argocd-server` Deployment and delete the ConfigMap:

```bash
make uninstall-ui-extension-local
```

Both targets are wrappers around `ui/dev/install-extension.sh`, run it with `-h`
for the available options.

!!! note
    This patches the `argocd-server` Deployment, which belongs to the Argo CD
    installation and is not managed by this project. It is a development helper.
    Production installation with the
    [argocd-extension-installer](https://github.com/argoproj-labs/argocd-extension-installer)
    init container is not implemented yet.

## Troubleshooting

The panel item does not appear:

* Reload the page. `/extensions.js` is fetched during the initial page
  rendering, so a running UI session will not pick up a newly installed
  extension.

* Check that `argocd-server` serves the extension. It prefixes every extension
  it loads with a `// source:` comment, so listing those comments shows exactly
  which files it picked up:

    ```bash
    kubectl -n argocd port-forward svc/argocd-server 8080:443
    curl -s http://localhost:8080/extensions.js | grep '^// source:'
    ```

    Use `curl -sk https://localhost:8080/...` instead if `argocd-server` is not
    running with `--insecure`. There should be exactly one line, mentioning
    `extension-monorepo-controller.js`. If there is none, `argocd-server` does
    not see the file; verify that it is mounted:

    ```bash
    kubectl -n argocd exec deploy/argocd-server -- \
        ls -l /tmp/extensions/resources/argocd-monorepo-controller
    ```

* Check the browser console. `argocd-server` wraps each extension in a
  `try`/`catch` block, so an extension that fails to load reports it only there,
  as `Extension extension-monorepo-controller.js failed to load: ...`.
