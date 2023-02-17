# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo 'Are you sure? [y/N]' && read ans && [ $${ans:-N} = y ]

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## tidy: to tidy and vendor dependencies
.PHONY: tidy
tidy:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Vendoring dependencies...'
	go mod vendor

## audit: format, vet and test code
.PHONY: audit
audit:
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

# ==================================================================================== #
# Build binaries
# ==================================================================================== #

current_time = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
git_description = $(shell git describe --always --dirty --tags --long)

## eagi/agent: build agent binary
.PHONY: eagi/agent
eagi/agent:
	env GOOS=linux GOARCH=amd64 \
		go build -o ./app/goEagi/${git_description}/Agent \
			-ldflags "-s -w \
			-X main.actor=agent \
			-X main.version=${git_description} \
			-X main.buildTime=${current_time}" \
			./app/goEagi/

## eagi/customer: build customer binary
.PHONY: eagi/customer
eagi/customer:
	env GOOS=linux GOARCH=amd64 \
		go build -o ./app/goEagi/${git_description}/Customer \
			-ldflags "-s -w \
			-X main.actor=customer \
			-X main.version=${git_description} \
			-X main.buildTime=${current_time}" \
			./app/goEagi/

## eagi/both: build agent & customer binaries
.PHONY: eagi/both
eagi/both:
	env GOOS=linux GOARCH=amd64 \
    		go build -o ./app/goEagi/${git_description}/Agent \
    			-ldflags "-s -w \
    			-X main.actor=agent \
    			-X main.version=${git_description} \
    			-X main.buildTime=${current_time}" \
    			./app/goEagi/
	env GOOS=linux GOARCH=amd64 \
    		go build -o ./app/goEagi/${git_description}/Customer \
    			-ldflags "-s -w \
    			-X main.actor=customer \
    			-X main.version=${git_description} \
    			-X main.buildTime=${current_time}" \
    			./app/goEagi/