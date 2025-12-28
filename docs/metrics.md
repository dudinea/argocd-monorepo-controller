# Monorepo Controller Metrics

Both Monorepo Controller and Monorepo Repo Server expose Prometheus metrics


## Prometheus Operator

If using Prometheus Operator, the following ServiceMonitor [example
manifests](https://github.com/argoproj-labs/argocd-monorepo-controller/tree/main/samples/metrics) can be used.  Add a namespace where Argo CD is installed
and change `metadata.labels.release` to the name of label selected by
your Prometheus.

```yaml
{!docs/argocd-monorepo-controller-sm.yaml!}
```

```yaml
{!docs/argocd-monorepo-controller-sm.yaml!}
```



