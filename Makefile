
default: test
all: test e2e

mod:
	go mod tidy
	go mod vendor

fmt: mod
	go fmt ./...

.PHONY: test
test: fmt
	go test -v ./...

build:
	go build -o ${BINARY} .

PHONY: e2e
e2e:
	docker-compose -f docker-compose.test.yml down --remove-orphans
	rm -vf testdata/*.generated testdata/*/*.generated
	docker-compose -f docker-compose.test.yml up --attach-dependencies --remove-orphans --renew-anon-volumes --force-recreate --always-recreate-deps --build --exit-code-from app --timeout 600
	grep -q newest testdata/manifest.json.generated
	grep -q oldest testdata/manifest.json.generated
	grep -q static testdata/manifest.json.generated
