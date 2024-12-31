<img src="https://raw.githubusercontent.com/matveynator/chicha-whois/refs/heads/master/chicha-whois-logo.png" alt="chicha-whois" width="50%" align="right" />


# chicha-whois

**chicha-whois** — это небольшая, но мощная CLI-утилита на Go для работы с базой RIPE NCC и генерации ACL (Access Control Lists) для BIND / списков для OpenVPN. Предельно проста в использовании и решает необходимые задачи.

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
sudo curl -L https://files.zabiyaka.net/chicha-whois/latest/no-gui/linux/amd64/chicha-whois -o /usr/local/bin/chicha-whois && \
sudo chmod +x /usr/local/bin/chicha-whois && \
/usr/local/bin/chicha-whois --version
```

### macOS (Intel)

```bash
sudo curl -L https://files.zabiyaka.net/chicha-whois/latest/no-gui/mac/amd64/chicha-whois -o /usr/local/bin/chicha-whois && \
sudo chmod +x /usr/local/bin/chicha-whois && \
/usr/local/bin/chicha-whois --version
```

### macOS (Apple Silicon, ARM64)

```bash
sudo curl -L https://files.zabiyaka.net/chicha-whois/latest/no-gui/mac/arm64/chicha-whois -o /usr/local/bin/chicha-whois && \
sudo chmod +x /usr/local/bin/chicha-whois && \
/usr/local/bin/chicha-whois --version
```

Проверить установку:
```bash
chicha-whois -h
```

---

## Основные команды

Ниже список ключевых опций:

| **Опция**                                     | **Описание**                                                                                                                                                                                                                       |
|-----------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `-u`                                          | Загрузить / обновить локальную базу RIPE NCC (скачать файл `ripe.db.inetnum.gz` и распаковать его).                                                                                                                                |
| `-dns-acl COUNTRYCODE`                        | Сгенерировать ACL для BIND (например, `RU`), сохранив в файл `acl_<COUNTRYCODE>.conf` в вашей домашней папке.                                                                                                                      |
| `-dns-acl-f COUNTRYCODE`                      | То же самое, но **с фильтрацией** вложенных подсетей (меньше записей).                                                                                                                                                              |
| `-ovpn COUNTRYCODE`                           | Создать список маршрутов для OpenVPN (exclude-route) в файле `openvpn_exclude_<COUNTRYCODE>.txt` (без фильтрации).                                                                                                                  |
| `-ovpn-f COUNTRYCODE`                         | Аналогично, но с фильтрацией вложенных сетей.                                                                                                                                                                                      |
| `-l`                                          | Показать список доступных кодов стран (упорядоченный по названию).                                                                                                                                                                 |
| `-h` или `--help`                             | Вывести справку и краткую информацию об опциях.                                                                                                                                                                                    |
| `-v` или `--version`                          | Показать версию приложения.                                                                                                                                                                                                         |
| **`-search [-dns | -ovpn | -ovpn-push] CC:kw1,kw2...`** | **Расширенный поиск.** Позволяет искать по коду страны (опционально) **и/или** по ключевым словам (`kw1`, `kw2` и т.д.). *Всегда* фильтрует вложенные подсети и **выводит** результат прямо в консоль (не в файл) в одном из форматов: DNS/OVPN/OVPN (push). |

### О команде `-search`

- **Формат вызова**:
  ```
  chicha-whois -search [-dns | -ovpn | -ovpn-push] CC:слово1,слово2,слово3
  ```
  Где `CC` — (опциональный) код страны, а после двоеточия — список ключевых слов, разделённых запятыми.

- Примеры:
  ```bash
  # Ищем в стране RU блоки, содержащие хотя бы одно из слов "ok.ru", "vk.ru", "drive2.ru"
  # Выводим результат в формате DNS ACL прямо на экран (без сохранения в файл):
  chicha-whois -search -dns RU:ok.ru,vk.ru,drive2.ru
  ```

  ```bash
  # Ищем по любым странам (перед двоеточием пусто) ключевые слова: "google.com", "amazon", "cloudflare"
  # и печатаем как push-маршруты для OpenVPN (в серверном стиле):
  chicha-whois -search -ovpn-push :google.com,amazon,cloudflare
  ```

- Если перед двоеточием **пусто** (`:kw1,kw2`), поиск идёт во всех блоках независимо от `country:`.  
- Если указан код страны (`RU:...`), тогда ищем только в блоках, у которых `country: RU`.  
- Ключевые слова ищутся **без учёта регистра** во всех строчках блока (organization, address и т.д.).  
- **Фильтрация вложенных подсетей** всегда включена — итоговые CIDR не будут дублировать друг друга.  
- **Вывод** поддерживает три формата:
  - `-dns`: BIND-ACL в стиле  
    ```bind
    acl "RU" {
      1.2.3.0/24;
      2.3.4.0/16;
    };
    ```
  - `-ovpn`: клиентский вариант (`route <ip> <mask> net_gateway`)  
  - `-ovpn-push`: серверный вариант (`push "route <ip> <mask>"`)

> Если формат не указан, CIDR-диапазоны просто выводятся списком.


---

## Примеры использования

1. **Обновить базу RIPE**  
   ```bash
   chicha-whois -u
   ```
   Скачивает последнюю версию базы `ripe.db.inetnum.gz` и распаковывает её в `~/.ripe.db.cache/`.

2. **Создать ACL для России (DNS BIND)**  
   ```bash
   chicha-whois -dns-acl RU
   ```
   Получаете файл `acl_RU.conf` с полным списком IP-сетей для России (без фильтра).

3. **Оптимизированный ACL**  
   ```bash
   chicha-whois -dns-acl-f RU
   ```
   То же самое, но отфильтрованные подсети: меньше записей, нет вложенных.

4. **Список кодов стран**  
   ```bash
   chicha-whois -l
   ```
   Выводит, например, `RU - Russia`, `UA - Ukraine` и т.д.

5. **OpenVPN exclude-route для RU**  
   ```bash
   chicha-whois -ovpn RU
   ```
   Создаёт `openvpn_exclude_RU.txt` с маршрутами типа `route 1.2.3.0 255.255.255.0 net_gateway`.

6. **Фильтрация для OpenVPN**  
   ```bash
   chicha-whois -ovpn-f RU
   ```
   Аналогично, но удаляет вложенные / дублирующиеся сети.

7. **Поиск по стране и ключевым словам**  
   ```bash
   chicha-whois -search -dns UA:google.com,kyivstar,mts
   ```
   - Находит в базе `inetnum`-блоки, у которых `country: UA` + в тексте фигурируют хотя бы одно из слов (`google.com`, `kyivstar`, `mts`).
   - Фильтрует вложенные CIDR.
   - Печатает результат в BIND-стиле ACL сразу в консоль.

8. **Поиск без указания страны**  
   ```bash
   chicha-whois -search -ovpn-push :cloudflare,amazon
   ```
   - Игнорируем поле `country:` — берём любую страну, если в блоке есть `cloudflare` или `amazon`.
   - Результат выводится в формате `push "route ..."` для серверной части OpenVPN (тоже только на экран).

---

## Дополнительно

- **Путь к локальной базе**: `~/.ripe.db.cache/ripe.db.inetnum`  
- **DNS ACL** (опции `-dns-acl`/`-dns-acl-f`): сохраняются в `~/acl_<COUNTRYCODE>.conf`  
- **OpenVPN** (опции `-ovpn`/`-ovpn-f`): сохраняются в `~/openvpn_exclude_<COUNTRYCODE>.txt`  
- **Поиск** (`-search`): **не** пишет файлы, а выводит всё в консоль.

---

## Пример настройки BIND9

Допустим, вы создали ACL-файлы для RU и UA:

```
include "/etc/bind/acl_RU.conf";
include "/etc/bind/acl_UA.conf";

view "Russia" {
    match-clients { RU; };  # RU
    zone "domain.com" {
        type master;
        file "/etc/bind/zones/db.domain.com.RU";
    };
};

view "Ukraine" {
    match-clients { UA; };  # UA
    zone "domain.com" {
        type master;
        file "/etc/bind/zones/db.domain.com.UA";
    };
};

view "default" {
    match-clients { any; };
    zone "domain.com" {
        type master;
        file "/etc/bind/zones/db.domain.com.default";
    };
};
```

- ACL-файлы можно положить, например, в `/etc/bind/acl_RU.conf` и `/etc/bind/acl_UA.conf`.
- Внутри каждого `acl_RU.conf` будет примерно:
  ```bind
  acl "RU" {
    100.43.64.0/24;
    100.43.65.0/24;
    ...
  };
  ```
- Не забудьте создать соответствующие zone-файлы:
  ```
  /etc/bind/zones/db.domain.com.RU
  /etc/bind/zones/db.domain.com.UA
  /etc/bind/zones/db.domain.com.default
  ```
- Проверка:
  ```
  sudo named-checkconf
  sudo systemctl restart bind9
  ```

---

## Пример настройки OpenVPN

Чтобы исключить маршруты целой страны из туннеля, можно добавить в `.ovpn` (клиентский конфиг) следующие строки:

```bash
# Пропускаем весь трафик через VPN:
redirect-gateway def1

# Исключаем (маршруты RU идут вне VPN):
route 100.43.64.0 255.255.255.0 net_gateway
route 100.43.65.0 255.255.255.0 net_gateway
...
```

Если нужно «толкать» такие маршруты с **серверной** стороны, используйте формат с `push "route ..."` — это можно сгенерировать через `-search -ovpn-push` или вручную дописать в серверный `server.conf`:

```bash
push "redirect-gateway def1"
push "route 100.43.64.0 255.255.255.0"
...
```

---

Наслаждайтесь **chicha-whois**! Если есть предложения или вопросы, создайте Issue или Pull Request в репозитории.
