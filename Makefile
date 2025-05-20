.PHONY: test

# run unit test
test:
	go test ./...

# generate coverage statistics.
cover:
	go test ./... -coverprofile cover.out

# generate coverage statistics and open it in the browser.
report: cover
	go tool cover -html=cover.out