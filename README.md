# kratix-backstage-generator-pipeline
Generate templates and components for your kratix promise automagically

## Usage
run `make build-and-load` to load the image into your platform cluster. Update
your promises to have:
```
              - image: ghcr.io/aclevername/kratix-backstage-generator-pipeline:v0.1.0
                name: backstage

```

in the promise and resource workflows
