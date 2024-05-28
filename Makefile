default: testacc

fmt:
	terraform fmt -recursive

gen:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

build: terraform-provider-coder

terraform-provider-coder: provider/*.go main.go
	CGO_ENABLED=0 go build .

# Run integration tests
.PHONY: test-integration
test-integration: terraform-provider-coder
	go test -v ./integration

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m
