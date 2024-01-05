IMG=ghcr.io/syntasso/kratix-backstage-generator-pipeline:v0.2.0

build-and-load:
	docker build --tag ${IMG} .
	kind load docker-image ${IMG} --name platform

test:
	ginkgo -r lib

release:
	if ! docker buildx ls | grep -q "kratix-image-builder"; then \
		docker buildx create --name kratix-image-builder; \
	fi;
	docker buildx build --builder kratix-image-builder --push --platform linux/arm64,linux/amd64 -t ${IMG} .
	docker push ${IMG}
