<img src="https://raw.githubusercontent.com/matveynator/chicha-whois/refs/heads/master/chicha-whois-logo.png" alt="chicha-whois" width="50%" align="right" />

# chicha-whois

**chicha-whois** — это небольшая, но мощная CLI-утилита на Go для работы с офлайн-базой RIPE NCC и генерации:
1. ACL (Access Control Lists) для DNS-сервера BIND  
2. Списков маршрутов для OpenVPN (exclude-route)  
3. Избирательных («точечных») поисков по IP-блокам для любых комбинаций [страна/ключевые слова].  

Особенно полезно, если нужно:
- Исключить **целую страну** или конкретные сервисы (по ключевым словам) из VPN-трафика (так называемая «раздельная маршрутизация», чтобы, например, сайт drive2.ru ходил напрямую, а остальной трафик — через VPN).  
- Автоматически вести «белые/чёрные» списки IP-адресов в BIND с учётом геолокации или поиска по организациям.  

Утилита умеет работать офлайн: один раз скачивает свежую базу `ripe.db.inetnum` из RIPE NCC, после чего все операции (генерация списков, поиск) происходят локально.

---

## Скачивание

Выберите нужную вам сборку:

- [Linux AMD64](https://files.zabiyaka.net/chicha-whois/latest/no-gui/linux/amd64/chicha-whois)  
- [Windows AMD64](https://files.zabiyaka.net/chicha-whois/latest/no-gui/windows/amd64/chicha-whois.exe)  
- [macOS (Intel) AMD64](https://files.zabiyaka.net/chicha-whois/latest/no-gui/mac/amd64/chicha-whois)  
- [Linux ARM64](https://files.zabiyaka.net/chicha-whois/latest/no-gui/linux/arm64/chicha-whois)  

Другие варианты смотрите в [каталоге всех бинарников](https://files.zabiyaka.net/chicha-whois/latest/no-gui/).

---

## Установка

### Linux (AMD64)

```bash
sudo curl -L https://files.zabiyaka.net/chicha-whois/latest/no-gui/linux/amd64/chicha-whois -o /usr/local/bin/chicha-whois
sudo chmod +x /usr/local/bin/chicha-whois
chicha-whois --version
```

### macOS (Intel)

```bash
sudo curl -L https://files.zabiyaka.net/chicha-whois/latest/no-gui/mac/amd64/chicha-whois -o /usr/local/bin/chicha-whois
sudo chmod +x /usr/local/bin/chicha-whois
chicha-whois --version
```

### macOS (Apple Silicon, ARM64)

```bash
sudo curl -L https://files.zabiyaka.net/chicha-whois/latest/no-gui/mac/arm64/chicha-whois -o /usr/local/bin/chicha-whois
sudo chmod +x /usr/local/bin/chicha-whois
chicha-whois --version
```

Проверить установку:
```bash
chicha-whois -h
```

---

## Основные возможности

1. **Локальная база RIPE NCC**  
   Для офлайн-работы вам нужно один раз скачать базу (ключ `-u`). Файл `ripe.db.inetnum.gz` скачивается и сохраняется в `~/.ripe.db.cache/`, а затем распаковывается.

2. **Генерация DNS ACL (BIND)**  
   Можно создать ACL-файл по коду страны (например, `RU`) — будет готовый список IP-сетей, чтобы прописать их в `named.conf`.  
   - Без фильтрации (ключ `-dns-acl COUNTRYCODE`)  
   - С фильтрацией вложенных CIDR (ключ `-dns-acl-f COUNTRYCODE`)  

3. **Генерация списков маршрутов для OpenVPN**  
   Аналогично:  
   - `-ovpn COUNTRYCODE` — полный список сетей, «выкладываем» (exclude-route) их из VPN-туннеля.  
   - `-ovpn-f COUNTRYCODE` — оптимизированный список (без вложенных подсетей).  
   - Результат пишется в файл `~/openvpn_exclude_<COUNTRYCODE>.txt`.  

4. **Точечный поиск (`-search`)**  
   Ищем IP-блоки в базе RIPE по коду страны **и/или** по ключевым словам.  
   - Пример: `chicha-whois -search -ovpn RU:ok.ru,drive2.ru` найдёт все блоки в `country: RU`, у которых встречаются слова «ok.ru» или «drive2.ru» в описаниях, организации и т.д.  
   - Можно вообще не указывать страну: `chicha-whois -search -ovpn-push :cloudflare,amazon` — ищем по всем странам, но только те блоки, где есть «cloudflare» или «amazon».  
   - Есть три формата вывода: `-dns` (BIND-ACL), `-ovpn` (OpenVPN-клиентский), `-ovpn-push` (OpenVPN-серверный).  

Все нужные CIDR-диапазоны вы получаете или в виде файла, или сразу в консоль (если используете `-search`).

---

## Таблица опций

| **Опция**                                     | **Описание**                                                                                                                           |
|-----------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------|
| `-u`                                          | Загрузить / обновить локальную базу RIPE NCC (скачивает `ripe.db.inetnum.gz` в `~/.ripe.db.cache/`, распаковывает).                  |
| `-dns-acl COUNTRYCODE`                        | Сгенерировать ACL для BIND (пример: `-dns-acl RU`) и сохранить в файл `acl_RU.conf` в домашнюю папку.                                |
| `-dns-acl-f COUNTRYCODE`                      | То же самое, но с фильтрацией вложенных подсетей (получается меньше записей).                                                         |
| `-ovpn COUNTRYCODE`                           | Создать список маршрутов для OpenVPN (exclude-route) и сохранить в файл `openvpn_exclude_RU.txt` (без фильтрации).                   |
| `-ovpn-f COUNTRYCODE`                         | Аналогично, но с фильтрацией вложенных сетей.                                                                                         |
| `-l`                                          | Показать список доступных кодов стран (упорядоченных по названию).                                                                   |
| `-h` или `--help`                             | Вывести справку (этот список).                                                                                                        |
| `-v` или `--version`                          | Показать версию приложения.                                                                                                           |
| `-search [-dns \| -ovpn \| -ovpn-push] CC:kw1,kw2,...` | Расширенный поиск по коду страны (опционально) **и/или** ключевым словам (нечувствительно к регистру).  Результат выводится в консоль. |

---

## Примеры использования

### 1. Обновить базу RIPE
```bash
chicha-whois -u
```
Теперь все дальнейшие команды работают локально, без внешних запросов к RIPE.

### 2. DNS ACL для России (BIND)
```bash
chicha-whois -dns-acl RU
```
Результат: файл `acl_RU.conf` со всеми IP-сетями (без фильтра).

### 3. Оптимизированный DNS ACL
```bash
chicha-whois -dns-acl-f RU
```
То же самое, но с вырезанием вложенных CIDR — выходит более компактный список.

### 4. Список кодов стран
```bash
chicha-whois -l
```
Например: `RU - Russia`, `UA - Ukraine`, `BY - Belarus`, `KZ - Kazakhstan`, и т.д.

### 5. OpenVPN exclude-route для RU
```bash
chicha-whois -ovpn RU
```
Сгенерирует `openvpn_exclude_RU.txt`, где каждая строка вида:
```
route 1.2.3.0 255.255.255.0 net_gateway
route 5.6.7.0 255.255.255.0 net_gateway
...
```
Вставьте это в ваш `.ovpn`, если хотите, чтобы трафик в эти сетки шёл напрямую (в обход VPN).

### 6. Фильтрация для OpenVPN
```bash
chicha-whois -ovpn-f RU
```
Аналогично п.5, но без дубликатов из-за вложенных подсетей.

### 7. Поиск по стране и ключевым словам
```bash
# Найдём IP-блоки в country: UA, где есть "google.com" или "kyivstar" или "mts", и выведем результат в стиле DNS-ACL (прямо в консоль):
chicha-whois -search -dns UA:google.com,kyivstar,mts
```
- Все найденные CIDR будут отфильтрованы от вложений.
- Вывод будет примерно в таком стиле:
  ```bind
  acl "UA" {
    91.200.0.0/16;
    91.202.128.0/19;
    ...
  };
  ```

### 8. Поиск **без указания страны**
```bash
# Ищем "cloudflare" или "amazon" во всех countries:
chicha-whois -search -ovpn-push :cloudflare,amazon
```
- Утилита найдёт все сети, где в `organization`, `descr`, `address` и пр. есть упоминание cloudflare/amazon.  
- Выведет результат в формате:
  ```
  push "route 104.16.0.0 255.255.240.0"
  push "route 13.32.0.0 255.255.0.0"
  ...
  ```

---

## Настройка OpenVPN для исключений

### Клиентский конфиг (пример)

Если хотите пускать **весь** трафик через VPN, но **исключить** какие-то IP-блоки (чтобы шли напрямую), добавьте в `.ovpn` (клиентский):

```bash
# Пропустить всё через VPN:
redirect-gateway def1

# Исключаем (пусть идёт вне VPN):
route 1.2.3.0 255.255.255.0 net_gateway
route 100.43.64.0 255.255.255.0 net_gateway
...
```
Такой список можно сгенерировать командой:
```bash
chicha-whois -ovpn RU
```
Либо, если хотите исключить лишь конкретные организации или сайты:
```bash
# Ищем в стране RU упоминание drive2, ok.ru:
chicha-whois -search -ovpn RU:drive2,ok.ru
```
Скопируйте строчки `route ... net_gateway` в клиентский `.ovpn`.

### Серверный конфиг (пример)

Если вам нужно **на сервере** OpenVPN «толкать» (push) маршруты клиентам, используйте формат `-ovpn-push`. Полученный вывод можно вставить в `server.conf`:

```bash
push "redirect-gateway def1"
push "route 100.43.64.0 255.255.255.0"
push "route 100.43.65.0 255.255.255.0"
...
```
Или по ключевым словам, например:
```bash
# Находим блоки, в которых фигурирует "amazon":
chicha-whois -search -ovpn-push :amazon
```
Скопируйте результат в конфиг сервера.

---

## Настройка BIND

### Пример

Если вы сгенерировали файлы `acl_RU.conf` и `acl_UA.conf` в домашней папке, перенесите их в `/etc/bind/` (или любой другой путь по вкусу) и используйте в `named.conf` или в отдельном файле конфигурации:

```bind
include "/etc/bind/acl_RU.conf";
include "/etc/bind/acl_UA.conf";

view "Russia" {
    match-clients { RU; };
    zone "example.com" {
        type master;
        file "/etc/bind/zones/db.example.com.RU";
    };
};

view "Ukraine" {
    match-clients { UA; };
    zone "example.com" {
        type master;
        file "/etc/bind/zones/db.example.com.UA";
    };
};

view "default" {
    match-clients { any; };
    zone "example.com" {
        type master;
        file "/etc/bind/zones/db.example.com.default";
    };
};
```
Внутри `acl_RU.conf` будет что-то вида:
```bind
acl "RU" {
    1.2.3.0/24;
    2.3.4.0/16;
    ...
};
```

---

## Зачем это нужно?

- **Разделение трафика в OpenVPN**. Например, вы хотите, чтобы *весь мир* шёл через VPN (для безопасности), но доступ к локальным ресурсам (внутри страны или конкретным сайтам) шёл напрямую. Утилита `chicha-whois` сгенерирует нужные `route ... net_gateway` или `push "route ..."` строки.  
- **Гео-ориентированный DNS**. BIND можно «разделить» на разные views по странам, перенаправляя российский трафик на один контент, украинский на другой, остальной на третий. Нужно лишь поддерживать ACL для RU, UA и т.д.  
- **Геолокационная фильтрация**. В корпоративных сетях бывает, что вы не хотите принимать DNS-запросы/соединения от IP определённых стран. Опять же, `chicha-whois` создаёт нужные ACL.  
- **Поиск IP-блоков по ключевым словам**. Если вы расследуете спам/атаки, иногда нужно найти все IP-диапазоны, упомянутые под каким-то `organization` или `descr`. Утилита позволяет (офлайн!) найти и отфильтровать нужные CIDR в RIPE-базе.

---

## Куда складываются файлы?

- **RIPE-база**: `~/.ripe.db.cache/ripe.db.inetnum`  
- **DNS ACL** (`-dns-acl`/`-dns-acl-f`): `~/acl_<COUNTRYCODE>.conf`  
- **OpenVPN** (`-ovpn`/`-ovpn-f`): `~/openvpn_exclude_<COUNTRYCODE>.txt`  
- **Поиск** (`-search`): вывод *только в консоль*, в файл не пишет.

---

## Итог

**chicha-whois** — очень удобный инструмент, если вам нужно:

1. Генерировать ACL для BIND (по странам или конкретным ключевым словам).  
2. Делать исключения (exclude-routes) для OpenVPN: «Routing by country» или «Routing by organization/keywords».  
3. Производить быстрый офлайн-поиск по базе RIPE: по стране, по имени организации, по адресу и т.д.  

Установка и использование просты, база скачивается один раз, дальше всё работает локально.  
Если есть вопросы или улучшения — открывайте Issue или Pull Request в [репозитории](https://github.com/matveynator/chicha-whois).  

Приятной работы с **chicha-whois**!
