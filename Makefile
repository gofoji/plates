lint:
	golangci-lint run --sort-results

test:
	go test ./...

cover:
	go test	-coverprofile cp.out ./...
	go tool cover -html=cp.out

tidy:
	go mod tidy

update:
	go get -u all

.PHONY:	lint update tidy cover test