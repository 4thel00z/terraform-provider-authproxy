default: terraform-provider-authproxy

install:
	go install
terraform-provider-authproxy: install
	cp /home/ransomware/go/bin/terraform-provider-authproxy ~/.terraform.d/plugins/terraform.local/local/authproxy/1.0.0/linux_amd64/terraform-provider-authproxy_v1.0.0

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m



