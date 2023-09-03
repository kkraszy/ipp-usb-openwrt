include $(TOPDIR)/rules.mk
PKG_NAME:=ipp-usb
PKG_VERSION:=0.9.23
PKG_RELEASE:=1
PKG_BUILD_DEPENDS:=golang/host
PKG_BUILD_DIR:=$(BUILD_DIR)/ipp-usb
include $(INCLUDE_DIR)/package.mk
include $(TOPDIR)/feeds/packages/lang/golang/golang-values.mk
MAKE_VARS = \
GOARCH=$(GO_ARCH)

define Package/ipp-usb
  SECTION:=utils
  CATEGORY:=Utilities
  TITLE:=ipp-usb
  PKGARCH:=all
  DEPENDS:=+libusb-1.0 +libavahi-client
endef

define Build/Prepare
	mkdir -p $(PKG_BUILD_DIR)
	$(CP) ./src/* $(PKG_BUILD_DIR)/
endef

define Package/ipp-usb/description
ipp-usb
endef

define Package/ipp-usb/install
	$(INSTALL_DIR) $(1)/usr/bin/
	$(INSTALL_BIN) $(PKG_BUILD_DIR)/ipp-usb $(1)/usr/bin/ipp-usb
	$(CP) ./src/* $(1)/
	$(CP) $(PKG_BUILD_DIR)/etc/ipp/ipp-usb.conf $(1)/etc/config/ipp-usb
endef

$(eval $(call BuildPackage,ipp-usb))