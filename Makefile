MAKEFLAGS += --silent

PLATFORM := $(shell uname -m)

ARCH :=
	ifeq ($(PLATFORM),x86_64)
		ARCH = amd64
	endif
	ifeq ($(PLATFORM),aarch64)
		ARCH = arm64
	endif
	ifeq ($(PLATFORM),armv7l)
		ARCH = armhf
	endif
GOARCH :=
	ifeq ($(ARCH),amd64)
		GOARCH = amd64
	endif
	ifeq ($(ARCH),i386)
		GOARCH = 386
	endif
	ifeq ($(ARCH),arm64)
		GOARCH = arm64
	endif
	ifeq ($(ARCH),armhf)
		GOARCH = arm
	endif

compile: test
ifeq ($(GOARCH),)
	$(error Invalid ARCH: $(ARCH))
endif
	go mod tidy
	go mod vendor > /dev/null 2>&1
	GOOS=linux GOARCH=$(GOARCH) go build

test:
	go test ./... -race -cover
