APP_NAME := screenmate-bot

GOOS ?= linux
GOARCH ?= amd64

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(APP_NAME) ./cmd/bot

.PHONY: run
run:
	go run ./cmd/bot

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: test
test:
	go test ./...

.PHONY: clean
clean:
	rm -f $(APP_NAME)

.PHONY: deploy
deploy: build
	scp $(APP_NAME) root@$(SERVER):/tmp/$(APP_NAME)

.PHONY: install
install:
	sudo systemctl stop screenmate-bot
	sudo cp $(APP_NAME) /opt/screenmate-bot/$(APP_NAME)
	sudo chown parking:parking /opt/screenmate-bot/$(APP_NAME)
	sudo chmod +x /opt/screenmate-bot/$(APP_NAME)
	sudo systemctl start screenmate-bot

.PHONY: restart
restart:
	sudo systemctl restart screenmate-bot

.PHONY: logs
logs:
	journalctl -u screenmate-bot -f

.PHONY: status
status:
	systemctl status screenmate-bot

.PHONY: package
package: build
	tar -czf screenmate-bot-linux-amd64.tar.gz screenmate-bot README.md
