default: testacc

fmt:
	terraform fmt -recursive

gen:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m