
## Configuring notifications

You can use the `mrp-controller.argoproj.io/change-revisions` (or `mrp-controller.argoproj.io/change-revision` if you only use
single source applications) to build triggers and templates for the Argo CD Notifications controller.


See sample triggers and templates in [samples/notifications](https://github.com/argoproj-labs/argocd-monorepo-controller/tree/main/samples/notifications) directory.  You may add sample triggers and templates from the
samples directory using a command like:

```shell
kubectl patch -n argocd cm argocd-notifications-cm --patch-file  samples/notifications/patch.yaml 
```

### The sample trigger

Triggers after Application syncing has succeeded and the Change Revision for the Application has changed.

```yaml
{!docs/trigger.yaml!}
```

### The sample template

The sample template uses the `mrp-controller.argoproj.io/change-revisions` annotation to build a table of all application sources
and their last Change Revisions.  Assuming that the Git repository is located on GitHub it gives links to the application manifests
in git and to the Change Revision commit.  For Helm Repository based sources it shows helm repository chart name and its version.

```yaml
{!docs/template.yaml!}
```



