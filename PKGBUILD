# Maintainer: André Dias <diaso.andre@outlook.com>

pkgname=shield-cli
pkgver=0.1.0
pkgrel=1
pkgdesc="Secure SSH credential management with Zero Plaintext Files"
arch=('x86_64' 'aarch64')
url="https://github.com/dias-andre/shield"
license=("GPL3")
depends=('glibc')
makedepends=('go' 'git')
source=("${pkgname}-${pkgver}.tar.gz::${url}/archive/refs/tags/${pkgver}.tar.gz")
sha256sums=('ad19f0f12af61e70e08b64940fbf964c2d3b9933bbed4ab616b8ca717f48001a')

build() {
  cd "shield-${pkgver}"

  export CGO_CPPFLAGS="${CPPFLAGS}"
  export CGO_CFLAGS="${CFLAGS}"
  export CGO_CXXFLAGS="${CXXFLAGS}"
  export CGO_LDFLAGS="${LDFLAGS}"
  export GOFLAGS="-buildmode=pie -trimpath -mod=readonly -modcacherw"

  go build -o build/shield .
}

package() {
  cd "shield-${pkgver}"

  install -Dm755 build/shield "$pkgdir/usr/bin/shield"
  install -d -m755 "$pkgdir/etc/shield"
}
