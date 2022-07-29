MAKEFLAGS += --silent

TARGET = hd-idle
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

ifdef DESTDIR
# dh_auto_install (Debian) sets this variable
  TARGET_DIR = $(DESTDIR)/usr
else
  TARGET_DIR ?= /usr/local
endif

all: $(TARGET)

distclean: clean

clean:
	rm -f $(TARGET)

install: $(TARGET)
	install -D -g root -o root $(TARGET) $(TARGET_DIR)/sbin/$(TARGET)
	install -D -g root -o root $(TARGET).8 $(TARGET_DIR)/share/man/man8/$(TARGET).8

$(TARGET):
ifeq ($(GOARCH),)
	$(error Invalid ARCH: $(ARCH))
endif
	GOOS=linux GOARCH=$(GOARCH) go build

test:
	go test ./... -race -cover
