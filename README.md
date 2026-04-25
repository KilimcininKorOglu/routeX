# RouteX

[English](README.en.md)

DNS tabanlı seçici trafik yönlendirme uygulaması. OpenWrt ve Entware (Keenetic) yönlendiriciler için tasarlandı.

## Ne Yapar

RouteX, yönlendiricinizdeki DNS sorgularını yakalayarak belirlediğiniz alan adlarına giden trafiği farklı ağ arayüzlerine yönlendirir. Örneğin, belirli servislerin trafiğini VPN tüneli üzerinden, geri kalan trafiği ise normal bağlantınız üzerinden gönderebilirsiniz.

Nasıl çalışır:

1. Mevcut DNS sunucusunun önüne bir ara katman yerleştirilir
2. Gelen DNS sorguları yakalanır ve yanıtlar önbelleğe alınır
3. IP adresleri ile alan adları eşleştirilir
4. Eşleşen trafik iptables/ipset kurallarıyla hedef arayüze yönlendirilir

İstemci tarafında DNS önbellek temizliğine gerek yoktur. Yalnızca servis yeniden başlatıldığında, önbellek ısınana kadar kısa bir bekleme süresi oluşur.

## Desteklenen Platformlar

| Platform              | Paket Yöneticisi | Paket Formatı |
| :-------------------- | :--------------- | :------------ |
| OpenWrt >= 25.12.X    | apk              | .apk          |
| OpenWrt <= 24.10.X    | opkg             | .ipk          |
| Entware (Keenetic)    | opkg             | .ipk          |

## Kurulum

Aşağıdaki komut platformunuzu ve mimarinizi otomatik algılayıp paket deposunu ekler:

```shell
wget -qO- https://raw.githubusercontent.com/KilimcininKorOglu/routex/develop/scripts/add_repo.sh | sh
```

Ardından paket yöneticinizle kurulumu tamamlayın:

**Entware (Keenetic):**

```shell
opkg update && opkg install routex
/opt/etc/init.d/S99routex start
```

**OpenWrt (opkg):**

```shell
opkg update && opkg install routex
service routex start
```

**OpenWrt (apk):**

```shell
apk update && apk add --allow-untrusted routex
service routex start
```

Güncelleme için aynı `opkg update && opkg install routex` veya `apk update && apk add routex` komutunu tekrarlayın.

## Kural Türleri

Gruplar içinde tanımladığınız kurallar, DNS sorgularının hangi arayüze yönlendirileceğini belirler. Dört kural türü desteklenir:

### Ad Alanı

Belirtilen alan adı ve tüm alt alan adlarını kapsar.

`example.com` kuralı ile:

```
example.com             eşleşir
sub.example.com         eşleşir
sub.sub.example.com     eşleşir
anotherexample.com      eşleşmez
example.net             eşleşmez
```

### Joker

`*` (sınırsız karakter) ve `?` (tek karakter) ile esnek eşleştirme.

`*example.com` kuralı ile:

```
example.com             eşleşir
sub.example.com         eşleşir
anotherexample.com      eşleşir
example.net             eşleşmez
```

### Alan Adı

Yalnızca tam eşleşen alan adına uygulanır, alt alan adları dahil değildir.

`sub.example.com` kuralı ile:

```
sub.example.com         eşleşir
example.com             eşleşmez
sub.sub.example.com     eşleşmez
```

### Düzenli İfade

İleri düzey kullanıcılar için. [dlclark/regexp2](https://github.com/dlclark/regexp2) motorunu kullanır.

`^[a-z]*example\.com$` kuralı ile:

```
example.com             eşleşir
anotherexample.com      eşleşir
sub.example.com         eşleşmez
```

## Web Arayüzü

Kurulumdan sonra varsayılan olarak `http://<yönlendirici-ip>:8080` adresinden erişebilirsiniz. Kimlik doğrulama varsayılan olarak açıktır; giriş için yönlendiricinin sistem kullanıcı bilgileri kullanılır:

| Platform           | Kullanıcı Adı | Şifre Kaynağı     |
| :----------------- | :------------- | :----------------- |
| OpenWrt            | `root`         | `/etc/shadow`      |
| Entware (Keenetic) | `root`         | `/opt/etc/shadow`  |

Entware kullanıcıları root şifresi belirlenmemişse `passwd` komutuyla şifre oluşturmalıdır.

Web arayüzü üzerinden:

- Grup oluşturma, düzenleme ve silme
- Kural ekleme, düzenleme ve sıralama
- Yapılandırma içeri/dışa aktarma
- Arama ve filtreleme
- Sistem ayarlarını görüntüleme

## Teknik Detaylar

| Özellik          | Değer                                              |
| :--------------- | :------------------------------------------------- |
| Dil              | Go 1.23                                            |
| Web Arayüzü      | templ + htmx + Alpine.js                           |
| DNS Motoru       | miekg/dns ile MITM proxy                           |
| Ağ Yönetimi      | iptables, ipset, netlink                           |
| Yapılandırma     | YAML                                               |
| Kimlik Doğrulama | JWT (varsayılan olarak açık)                      |
| Paket Formatı    | .ipk (opkg) ve .apk (Alpine)                      |
| Lisans           | GPL-3.0-or-later                                   |

## Derleme

Kaynak koddan derlemek için:

```shell
cp config/openwrt/aarch64_generic.config .config
make
```

Çıktı `.build/` dizinine yazılır. Derleme için Go 1.23, templ, upx ve fakeroot gereklidir.

## Lisans

Bu proje [GPL-3.0-or-later](LICENSE) lisansı altında dağıtılmaktadır.

## Atıf

Bu proje, [MagiTrickle](https://gitlab.com/magitrickle/magitrickle) projesinin bir çatalıdır. Orijinal proje Ponywka ve katkıda bulunanlar tarafından GPL-3.0-or-later lisansı altında geliştirilmiştir. Katkıda bulunanların tam listesi [CONTRIBUTORS.md](CONTRIBUTORS.md) dosyasında yer almaktadır.
