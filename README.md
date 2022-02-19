[![GitHub release](https://img.shields.io/github/release/cybozu-go/usocksd.svg?maxAge=60)][releases]
[![GoDoc](https://godoc.org/github.com/cybozu-go/usocksd?status.svg)][godoc]
[![CircleCI](https://circleci.com/gh/cybozu-go/usocksd.svg?style=svg)](https://circleci.com/gh/cybozu-go/usocksd)
[![Go Report Card](https://goreportcard.com/badge/github.com/cybozu-go/usocksd)](https://goreportcard.com/report/github.com/cybozu-go/usocksd)
[![License](https://img.shields.io/github/license/cybozu-go/usocksd.svg?maxAge=2592000)](LICENSE)

### Disclamer
This is a fork with SOCKS4[A] Patch for particular russian ISP to bypass glitching DPI thru reattempting request for uncensoring direct internet
You wont find anything interesting here for yourself unless you have that exact same ISP with bugged DPI

### Изменения

Изменены два файла
* server.go
* socks4.go

Пропатчен только SOCKS4/A код, тк он более простой и более тупой (нету двухсторонней неготиации, что упрощает патч). Думаю на 5ый сокс можно пока забить болт тк 4ый тупо быстрее. 5ый вообще не нужен если не нужна аунтификация или удпбинд, хз может потом сделаю но сначала надо на 4ом соксе сделать нормально чтобы работало

# DPI
* Провайдер снифает траффик, позволяет установить соединение
* В случае 80 порта, провайдер ждет когда в пакете появится GET domain.com/
* Проверяет домен, если там запрещенный - шлет другой ответ с редиректом на баннер РКН
* Если порт 443 - то ниче не анализирует а сразу шлет RST пакет

# Изначальный метод обхода DPI
Заключался в дропе посредством iptables - RST пакетов на 443 порту и анализе ответов на 80 порту и дропе если это редирект на баннер
Сейчас это работает но с перебоями, вообще помоему всегда было с перебоями просто я както не замечал потому что хром умеет сам долбится (а фф уже нет)

# Замеренные вероятности при использовании фильтра iptables
curl rutracker.org
50% - пройдет сразу
40% - на вторую попытку
9% - на третью
1% - на четвертую,пятую

# Концепт технологии анфака РКН на провайдере через сокс сервер
* Socks4/А сервер получает запрос
* Буферизует данные полученные от клиента
* Смотрит пытается считать ответ от рутрекера как пример
* Если ответ не получен - переотправляет из буффера и повторяет n раз пока ответ не придет
* В случае если получено хотябы 1 байт в ответе - сингналит клиентскому треду что бы переключился с буфера на чтение с настоящего клиента и продолжел работу

# Как именно это все реализовано на Go
* Вместо обычного чтения из сокета используется TeeReader который зеркалирует чтение в буффер
* В случае обрыва идет следующая итерация цикла и пересоздается TeeReader который эмулирует поток чтения для новой итерации
* Это все заправляется опять в функцию handlesocks4 которая уже на второй раз ниче не отвечает клиенту а тупо делает коннект новый (dial)
* Далее все опять же дальше эмулируется до тех пор пока мы не смогли считать ответ, если получен сигнал halfass - то переключаемся на обычное чтение с клиента

# Что работает
Работает вприцнипе все, даже TLSу нравится переотправленный хендшейк и переключение,значит никакие байты нигде не теряются

# Что я пытался сделать
Абстрагировать все потоки в channelы - нихуя не вышло из этого (да и хуевая идея в плане перфоманса), сокс сервер не делал исходящий коннект какогто хрена
Но именно поэтому там есть функция yobasync которая не исопльзуется
А функция handleConnectionPipe - это старый рабочий код который не позволяет сделать концепт, он либо висит либо закрывает клиентское соединение (а этого делать нельзя пока точно не закончили цикл долбления) 

# Проблемы
* Точно течет память, если отрубить сборщик мусора у меня повис комп через 1 минуту хотя оперативы 32гб
* Жрет процессор - скорее всего сборщик мусора который лечит то что иначе у меня комп повис бы

# Что можно улучшить
* Выкинуть нахуй этот TeeReader скорее всего он тормозит пиздец и заменить на байтоебство по буферу
* Либо оставить его но пропатчить/заменить что бы он использовал один буфер и можно было делать .Seek(0) или резет по буфферу, потому что щас изза невозможности перемотки на каждую итерацию приходится пересоздовать буффер.
Данные которые мы получаем первыми до того как получили ответ от сервера не меняются поэтому этот мой дроч можно сократить до одного буфера который просто пересчитывать по памяти уже потом
* Какимто образом абортить треды которые точно уже известно что не нужны (коннект к серверу рутрекера не вышел)
* Вместо пересоздания клиентского треда можно было бы сообщать както ему что коннект к серверу рутрекера поменялся, что бы он переключался на новые стримы
* Чистить память что бы сбощик мусора так не охуевал
* Переиспользовать память помимо того буффера что бы память так не текла


# Как запустить
* Качаешь последний гоу, добавляешь в PATH (на старых не работает)
* Клонируешь реп с гита
* Заходишь в папку, делаешь чето типа go get ./... - он подсосет все 
* Запускаешь go run cmd/usocksd/main.go
* Ctrl+C какойто обосранный из апстрима, иногда вообще не работает, его может стоит тоже хукнуть и прибивать себя через kill -9 так как очень больно тестить без нормального Ctrl+C

Micro SOCKS server
==================

**usocksd** is a SOCKS server written in Go.

[`usocksd/socks`](https://godoc.org/github.com/cybozu-go/usocksd/socks)
is a general purpose SOCKS server library.  usocksd is built on it.

Features
--------

* Support for SOCKS4, SOCKS4a, SOCK5

    * Only CONNECT is supported (BIND and UDP associate is missing).

* Graceful stop & restart

    * On SIGINT/SIGTERM, usocksd stops gracefully.
    * On SIGHUP, usocksd restarts gracefully.

* Access log

    Thanks to [`cybozu-go/log`](https://github.com/cybozu-go/log),
    usocksd can output access logs in structured formats including
    JSON.

* Specific network interface

    usocksd can be configured to use specific network interface
    for outgoing connections.

    It is extremely useful if you want to send all traffic to VPN/Wireguard device
    or you have multiple network cards.

* Multiple external IP addresses

    usocksd can be configured to use multiple external IP addresses
    for outgoing connections.

    usocksd keeps using the same external IP address for a client
    as much as possible.  This means usocksd can proxy passive FTP
    connections reliably.

    Moreover, you can use a [DNSBL][] service to exclude dynamically
    from using some undesirable external IP addresses.

* White- and black- list of sites

    usocksd can be configured to grant access to the sites listed
    in a white list, and/or to deny access to the sites listed in a
    black list.

    usocksd can block connections to specific TCP ports, too.

Install
-------

Use Go 1.7 or better.

```
go get -u github.com/cybozu-go/usocksd/...
```

Usage
-----

`usocksd [-h] [-f CONFIG]`

The default configuration file path is `/etc/usocksd.toml`.

In addition, `usocksd` implements [the common spec](https://github.com/cybozu-go/well#specifications) from [`cybozu-go/well`](https://github.com/cybozu-go/well).

usocksd does not have *daemon* mode.  Use systemd to run it on your background.

Configuration file format
-------------------------

`usocksd.toml` is a [TOML][] file.
All fields are optional.

```
[log]
filename = "/path/to/file"         # default to stderr.
level = "info"                     # critical, error, warning, info, debug
format = "plain"                   # plain, logfmt, json

[incoming]
port = 1080
addresses = ["127.0.0.1"]          # List of listening IP addresses
allow_from = ["10.0.0.0/8"]        # CIDR network or IP address

[outgoing]
allow_sites = [                    # List of FQDN to be granted.
    "www.amazon.com",              # exact match
    ".google.com",                 # subdomain match
]
deny_sites = [                     # List of FQDN to be denied.
    ".2ch.net",                    # subdomain match
    "bad.google.com",              # deny a domain of *.google.com
    "",                            # "" matches non-FQDN (IP) requests.
]
deny_ports = [22, 25]              # Black list of outbound ports
iface = tun0                       # Outgoing traffic binds to specific network interface
addresses = ["12.34.56.78"]        # List of source IP addresses
dnsbl_domain = "some.dnsbl.org"    # to exclude black listed IP addresses
```

Tuning
------

If you see usocksd consumes too much CPU, try setting [`GOGC`][GOGC] to higher value, say **300**.

License
-------

[MIT](https://opensource.org/licenses/MIT)

[releases]: https://github.com/cybozu-go/usocksd/releases
[DNSBL]: https://en.wikipedia.org/wiki/DNSBL
[TOML]: https://github.com/toml-lang/toml
[godoc]: https://godoc.org/github.com/cybozu-go/usocksd
[GOGC]: https://golang.org/pkg/runtime/#pkg-overview
