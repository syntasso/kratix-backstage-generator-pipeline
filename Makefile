build-and-load:
	docker build --tag ghcr.io/syntasso/kratix-backstage-generator-pipeline:v0.1.0 .
	kind load docker-image ghcr.io/syntasso/kratix-backstage-generator-pipeline:v0.1.0 --name platform

test:
	ginkgo -r lib
