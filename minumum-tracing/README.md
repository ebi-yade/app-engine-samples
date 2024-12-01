_**IMPORTANT NOTE**_: This app does not work as expected, because app engine does not support Ops Agent.

https://cloud.google.com/monitoring/agent/ops-agent#supported_vms

You may want to use `exporter/trace` of [opentelemetry-operations-go](https://github.com/GoogleCloudPlatform/opentelemetry-operations-go) instead.

# minimum-tracing

## Deployment

```sh
gcloud app deploy --version=$(git rev-parse --short HEAD) --quiet app.yaml
```