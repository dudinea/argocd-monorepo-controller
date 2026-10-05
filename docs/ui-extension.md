# UI Extension

The Monorepo Controller ships an [Argo CD UI
extension](https://argo-cd.readthedocs.io/en/latest/developer-guide/extensions/ui-extensions/)
that adds a **Change Revision** item to the Application status panel,
the bar at the top of the application view. It shows the commit that
actually changed the application's manifests, next to the repository
revision Argo CD displays under **Sync Status**.

## What is displayed

The item shows the [Change Revision](terminology.md#change-revision) from the
`mrp-controller.argoproj.io/change-revision(s)` annotations:

* **Single source applications**: the change revision, abbreviated to seven
  characters when it is a full commit SHA, and linked to the commit. Hover over it
  to see the full value.
* **Multi source applications**: one row per source, in the order the sources are
  declared in the application, each labelled with the source `name` if it has one
  and with the repository name otherwise.
* **Helm repository sources**: the chart version instead of a commit, since that
  is what the controller records for them.
* A source whose change revision has not been calculated yet shows as
  `—`. This is normal for an application that has not been synced yet.

The item is **not shown at all** for Applications without the
  `argocd.argoproj.io/manifest-generate-paths` annotation, which the
  controller skips entirely.

### Commit links

Revisions link to the commit in the repository, using the same logic
as Argo CD's own panels, so only GitHub, `gitlab.com`,`bitbucket.org`
and Bitbucket Server are supported.  On any other host the revision is
shown as plain text. 


### Commit details

The item has an ellipsis button, like the built in ones. It opens a panel showing, for
each application source:

* the repository, path (or chart) and target revision,
* the full change revision,
* and that commit's author, date, tags, signature and message.

Helm repository sources show their chart version and are not looked up, since a chart
version is not a commit.

## How it works

The change revisions themselves are read from the Application object the UI already
has. Only the commit details in the flyout need a request, to the Argo CD API, so
the extension never talks to the Monorepo Controller or its repo server.

## Installing with argocd-extension-installer

Every release publishes the extension as a an asset `extension.tar.gz`, 
which is meant to be installed into the Argo CD Server pod  using the 
[argocd-extension-installer](https://github.com/argoproj-labs/argocd-extension-installer).

Add the init container to the `argocd-server` Deployment, pointing it at the
release assets:

```yaml
spec:
  template:
    spec:
      initContainers:
        - name: monorepo-ui-extension
          image: quay.io/argoprojlabs/argocd-extension-installer:v0.0.9
          env:
            - name: EXTENSION_NAME
              value: monorepo-controller
            - name: EXTENSION_VERSION
              value: <version>
            - name: EXTENSION_URL
              value: https://github.com/argoproj-labs/argocd-monorepo-controller/releases/download/<VERSION>/extension.tar.gz
            - name: EXTENSION_CHECKSUM_URL
              value: https://github.com/argoproj-labs/argocd-monorepo-controller/releases/download/<VERSION>/checksums.txt
          volumeMounts:
            - name: extensions
              mountPath: /tmp/extensions/
          securityContext:
            runAsUser: 1000
            allowPrivilegeEscalation: false
      containers:
        - name: argocd-server
          volumeMounts:
            - name: extensions
              mountPath: /tmp/extensions/
      volumes:
        - name: extensions
          emptyDir: {}
```

`EXTENSION_CHECKSUM_URL` is optional. When it is set, the installer looks the
archive's file name up in the fetched file, so the last path segment of
`EXTENSION_URL` has to match an entry in it.


## Building

The sources live in `ui/src` and are bundled by [esbuild](https://esbuild.github.io/)
into a single file, `ui/dist/extension-monorepo-controller.js`. That bundle is
committed, so installing the extension needs nothing but `kubectl`.

After changing anything under `ui/src`, rebuild the bundle and commit it:

```bash
make build-ui-local
```

`make verify-ui-dist-local` fails if the committed bundle does not match the
sources, and `make test-ui-local` rebuilds before running the tests.

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


## Running Tests

The rendering logic has a dependency free test harness that runs the extension
against a stub React and a set of application fixtures, including the multi
source and Helm chart cases that are awkward to reproduce in a cluster:

```bash
make test-ui-local
```


## Troubleshooting

The panel item does not appear:

* Reload the page. `/extensions.js` is fetched during the initial page
  rendering, so a running UI session will not pick up a newly installed
  extension.

* Check that `argocd-server` serves the extension. It prefixes every extension
  it loads with a `// source:` comment, so listing those comments shows exactly
  which files it picked up:

    ```bash
    curl -s https://<HOSTNAME>:<PORT>/extensions.js | grep '^// source:'
    ```

  There should be exactly one line, mentioning `extension-monorepo-controller.js`. 
  If there is none, `argocd-server` does not see the file; verify that it is mounted:

    ```bash
    kubectl -n argocd exec deploy/argocd-server -- \
        ls -l /tmp/extensions/resources/argocd-monorepo-controller
    ```

* Check the browser console. `argocd-server` wraps each extension in a
  `try`/`catch` block, so an extension that fails to load reports it only there,
  as `Extension extension-monorepo-controller.js failed to load: ...`.
