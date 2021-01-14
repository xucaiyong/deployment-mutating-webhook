WEBHOOK_SERVICE?=hello-webhook-service
NAMESPACE?=default
CONTAINER_REPO?=quay.io/didil/hello-webhook
CONTAINER_VERSION?=0.1.9
CONTAINER_IMAGE=$(CONTAINER_REPO):$(CONTAINER_VERSION)

.PHONY: docker-build
docker-build:
	docker build -t $(CONTAINER_IMAGE) webhook

.PHONY: docker-push
docker-push:
	docker push $(CONTAINER_IMAGE) 

.PHONY: k8s-deploy
k8s-deploy: k8s-deploy-other k8s-deploy-csr k8s-deploy-deployment

.PHONY: k8s-deploy-other
k8s-deploy-other:
	kubectl apply -k k8s/other
	kubectl apply -k k8s/csr
	@echo Waiting for cert creation ...
	@sleep 15
	kubectl certificate approve $(WEBHOOK_SERVICE).$(NAMESPACE)

.PHONY: k8s-deploy-csr
k8s-deploy-csr:
	kubectl apply -k k8s/csr
	@echo Waiting for cert creation ...
	@sleep 15
	kubectl certificate approve $(WEBHOOK_SERVICE).$(NAMESPACE)

.PHONY: k8s-deploy-deployment
k8s-deploy-deployment:
#	kubectl kustomize set image CONTAINER_IMAGE=$(CONTAINER_IMAGE) k8s/deployment
	kubectl apply -k k8s/deployment

.PHONY: k8s-delete-all
k8s-delete-all:
	kubectl delete --ignore-not-found=true -k k8s/other
	kubectl delete --ignore-not-found=true -k k8s/csr 
	kubectl delete --ignore-not-found=true -k k8s/deployment 
	kubectl delete --ignore-not-found=true csr $(WEBHOOK_SERVICE).$(NAMESPACE)
	kubectl delete --ignore-not-found=true secret hello-tls-secret

.PHONY: test
test:
	cd webhook && go test ./...
