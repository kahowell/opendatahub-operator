Observability PoC Notes
-----------------------

The PoC installs into the OpenShift AI monitoring namespace `redhat-ods-monitoring`, so you'll probably need to install
OpenShift AI first (or adjust the namespace).

```shell
kubectl apply -k hack/observability
```

Some metrics are generated via quay.io/prometheuscommunity/avalanche.

Some port-forwards are necessary to interact with some components:

* OTLP/HTTP: `kubectl port-forward -n redhat-ods-monitoring deployments/otel-collector-collector 4318:4318`
* Grafana LGTM (serves as a sample 3rd party observability stack) `kubectl port-forward -n redhat-ods-monitoring deployments/lgtm 3000:3000`
* Prometheus: `kubectl port-forward -n redhat-ods-monitoring prometheus-odh-monitoring-stack-0 9090:9090`

Jaeger backed by Tempo is available at a route, use `kubectl get route` to determine the endpoint.

You can generate a span by port-forwarding OTLP/HTTP above and following directions here:

https://www.honeycomb.io/blog/test-span-opentelemetry-collector
