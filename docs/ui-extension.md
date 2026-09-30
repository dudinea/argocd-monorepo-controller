# UI Extension

The Monorepo Controller ships an [Argo CD UI extension](https://argo-cd.readthedocs.io/en/latest/developer-guide/extensions/ui-extensions/)
that adds a **Change Revision** item to the Application status panel, the bar at
the top of the application view where the sync status is shown.

It saves looking the value up with `kubectl`: the panel shows the commit that
actually changed the application's manifests, next to the repository revision
Argo CD displays under **Sync Status**.

## What is displayed

The item shows the [Change Revision](terminology.md#change-revision) from the
`mrp-controller.argoproj.io/change-revision(s)` annotations:

* **Single source applications**: the change revision, abbreviated to seven
  characters when it is a full commit SHA. Hover over it to see the full value.
* **Multi source applications**: one row per source, in the order the sources are
  declared in the application, each labelled with the source `name` if it has one
  and with the repository name otherwise.
* **Helm repository sources**: the chart version instead of a commit, since that
  is what the controller records for them.
* Values that are not commit SHAs, such as a branch name or a tag, are shown
  as they are.
* A source whose change revision has not been calculated yet shows as
  `—`. This is normal for an application that has not been synced yet.

The item is **not shown at all** for applications the controller does not track:

* Applications without the `argocd.argoproj.io/manifest-generate-paths`
  annotation, which the controller skips entirely.
* Applications managed by an ApplicationSet, unless the four
  `mrp-controller.argoproj.io/*` annotations are listed in
  `applicationsetcontroller.global.preserved.annotations`, otherwise the
  ApplicationSet controller removes them. See
  [ApplicationSet controller configuration](applicationsets.md).

## Commit details

The item has an ellipsis button, like the built in ones. It opens a panel showing, for
each application source:

* the repository, path (or chart) and target revision,
* the full change revision,
* and that commit's author, date, tags, signature and message.

The commit details are read from the Argo CD API, which enforces the same permission
needed to view the application, so no extra RBAC configuration is required. If the
request fails, for example because the session expired or the repository server cannot
resolve the revision, the reason is shown in place of the commit details and the rest of
the panel still renders.

Helm repository sources show their chart version and are not looked up, since a chart
version is not a commit.

!!! note
    The panel shows the details of the change revision itself, not the list of commits
    between the previously known revision and it. The Argo CD API returns metadata for
    one revision at a time and has no endpoint for a commit range.

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

The rendering logic has a dependency free test harness that runs the extension
against a stub React and a set of application fixtures, including the multi
source and Helm chart cases that are awkward to reproduce in a cluster:

```bash
make test-ui-local
```

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
