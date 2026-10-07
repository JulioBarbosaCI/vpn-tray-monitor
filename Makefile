# Compilacao cruzada de Linux para Windows. Sem CGO o binario nao depende de
# nenhuma DLL alem das do proprio Windows.
BUILD    := build
LDFLAGS  := -s -w
GOENV    := GOOS=windows GOARCH=amd64 CGO_ENABLED=0

.PHONY: all build test lint icons clean

all: test build

# Gera os dois binarios a partir da mesma fonte:
#  - vpnmon.exe    aplicacao grafica, o monitor residente (nao abre console)
#  - vpnmonctl.exe versao console, para -check e -set-password com saida legivel
build: icons
	@mkdir -p $(BUILD)
	$(GOENV) go build -trimpath -ldflags "$(LDFLAGS) -H=windowsgui" -o $(BUILD)/vpnmon.exe    ./cmd/vpnmon
	$(GOENV) go build -trimpath -ldflags "$(LDFLAGS)"               -o $(BUILD)/vpnmonctl.exe ./cmd/vpnmon
	@cp -n exemplos/vpn-nativa-windows.json $(BUILD)/config.json 2>/dev/null || true
	@ls -lh $(BUILD)/*.exe

test:
	go test ./... -cover

lint:
	gofmt -l . | tee /dev/stderr | (! read)
	go vet ./...
	GOOS=windows go vet ./...

icons:
	@go run ./tools/geniconos >/dev/null
	@cp assets/*.ico internal/trayui/icons/

clean:
	rm -rf $(BUILD)
