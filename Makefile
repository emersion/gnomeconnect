all:
	go build gnomeconnect.go
start:
	go run gnomeconnect.go
install-user:
	go install
	desktop-file-install --dir=$(HOME)/.local/share/applications gnomeconnect.desktop

.PHONY: all start install-user
