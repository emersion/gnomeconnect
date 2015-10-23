all:
	go build gnomeconnect.go
start:
	go run gnomeconnect.go

.PHONY: all start
