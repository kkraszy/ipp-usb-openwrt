include $(TOPDIR)/rules.mk
PKG_NAME:=ipp-usb
PKG_VERSION:=0.9.23
PKG_RELEASE:=1
PKG_BUILD_DEPENDS:=golang/host
PKG_BUILD_DIR:=$(BUILD_DIR)/ipp-usb
PKG_BUILD_FLAGS:=no-mips16


AGH_BUILD_TIME:=$(shell date -d @$(SOURCE_DATE_EPOCH) +%FT%TZ%z)
AGH_VERSION_PKG:=github.com/OpenPrinting/ipp-usb/internal/version
GO_PKG_LDFLAGS_X:=$(AGH_VERSION_PKG).channel=release \
	$(AGH_VERSION_PKG).version=$(PKG_SOURCE_VERSION) \
	$(AGH_VERSION_PKG).buildtime=$(AGH_BUILD_TIME) \
	$(AGH_VERSION_PKG).goarm=$(GO_ARM) \
	$(AGH_VERSION_PKG).gomips=$(GO_MIPS)


include $(INCLUDE_DIR)/package.mk
include $(TOPDIR)/feeds/packages/lang/golang/golang-package.mk
include $(TOPDIR)/feeds/packages/lang/golang/golang-values.mk

define Package/ipp-usb
  SECTION:=utils
  CATEGORY:=Utilities
  TITLE:=ipp-usb
  PKGARCH:=all
  DEPENDS:=$(GO_ARCH_DEPENDS) +libusb-1.0 +libavahi-client +libavahi-compat-libdnssd +avahi-utils
endef


MAKE_VARS += \
	GO_INSTALL_BIN_PATH="$(strip $(GO_PKG_INSTALL_BIN_PATH))" \
	BUILD_DIR="$(PKG_BUILD_DIR)" \
	GO_BUILD_DIR="$(GO_PKG_BUILD_DIR)" \
	GO_BUILD_BIN_DIR="$(GO_PKG_BUILD_BIN_DIR)" \
	GO_BUILD_DEPENDS_PATH="$(GO_PKG_BUILD_DEPENDS_PATH)" \
	GO_BUILD_DEPENDS_SRC="$(GO_PKG_BUILD_DEPENDS_SRC)" \
	GOOS="$(GO_OS)" \
	GOARCH="$(GO_ARCH)" \
	CC="$(TARGET_CC)" \
	CXX="$(TARGET_CXX)" \
	CGO_CFLAGS="$(filter-out $(GO_CFLAGS_TO_REMOVE),$(TARGET_CFLAGS))" \
	CGO_CPPFLAGS="$(TARGET_CPPFLAGS)" \
	CGO_CXXFLAGS="$(filter-out $(GO_CFLAGS_TO_REMOVE),$(TARGET_CXXFLAGS))" \
	CGO_LDFLAGS="$(TARGET_LDFLAGS)" \
	GOPATH="$(GO_PKG_BUILD_DIR)" \
	GOCACHE="$(GO_BUILD_CACHE_DIR)" \
	GOMODCACHE="$(GO_MOD_CACHE_DIR)" \
	GOFLAGS="$(GO_PKG_GCFLAGS)" \
	GO_PKG_CFLAGS="$(GO_PKG_CFLAGS)" \
	CGO_ENABLED=1 \
	GOENV=off \
	PREFIX=/usr \
	LIBEXECDIR=/usr/lib \
	SHAREDIR_CONTAINERS=/usr/share/containers \
	ETCDIR=/etc \
	BUILDTAGS="$(GO_PKG_TAGS)" \
	EXTRA_LDFLAGS="$(GO_PKG_LDFLAGS)"

define Build/Prepare
	mkdir -p $(PKG_BUILD_DIR)
	$(CP) ./src/* $(PKG_BUILD_DIR)/
endef

ifneq ($(CONFIG_USE_MUSL),)
  TARGET_CFLAGS += -D_LARGEFILE64_SOURCE
endif


define Package/ipp-usb/description
ipp-usb
endef


define Package/ipp-usb/install
	$(INSTALL_DIR) $(1)/usr/bin
	$(INSTALL_BIN) $(PKG_BUILD_DIR)/ipp-usb $(1)/usr/bin/ipp-usb
	$(INSTALL_DIR) $(1)/etc/init.d
	$(INSTALL_BIN) ./files/ipp-usb.init $(1)/etc/init.d/ipp-usb
	$(INSTALL_DIR) $(1)/etc/ipp-usb
	$(INSTALL_DATA) ./files/ipp-usb.conf $(1)/etc/ipp-usb/ipp-usb.conf
endef

$(eval $(call BuildPackage,ipp-usb))