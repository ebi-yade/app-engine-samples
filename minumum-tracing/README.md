## Deployment

```sh
gcloud app deploy --version=$(git rev-parse --short HEAD) --quiet app.yaml
```