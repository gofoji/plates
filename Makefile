lint:
	golangci-lint run 

fmt:
	golangci-lint fmt

test:
	go test -race ./...

cover:
	go test	-coverprofile cp.out ./...
	go tool cover -html=cp.out

tidy:
	go mod tidy

update:
	go get -u ./...

.PHONY:	lint update tidy cover test fmt
