<img src="https://raw.githubusercontent.com/matveynator/chicha-whois/refs/heads/master/chicha-whois-logo.png" alt="chicha-whois" width="50%" align="right" />

**chicha-whois** is a tiny but powerful CLI tool for working with the RIPE database and generating DNS ACLs. Clean, simple, and gets the job done. Written in Golang.

---

## Downloads

Pick your binary and get started:

- [Linux AMD64](https://files.zabiyaka.net/chicha-whois/latest/no-gui/linux/amd64/chicha-whois)  
- [Windows AMD64](https://files.zabiyaka.net/chicha-whois/latest/no-gui/windows/amd64/chicha-whois.exe)  
- [macOS AMD64](https://files.zabiyaka.net/chicha-whois/latest/no-gui/mac/amd64/chicha-whois)  
- [Linux ARM64](https://files.zabiyaka.net/chicha-whois/latest/no-gui/linux/arm64/chicha-whois)  

Need something else? [Check all binaries](https://files.zabiyaka.net/chicha-whois/latest/no-gui/).

---

## Installation

On Linux AMD64:
```bash
sudo curl -L https://files.zabiyaka.net/chicha-whois/latest/no-gui/linux/amd64/chicha-whois -o /usr/local/bin/chicha-whois && sudo chmod +x /usr/local/bin/chicha-whois; /usr/local/bin/chicha-whois --version;
```

On macOS (Intel):
```bash
sudo curl -L https://files.zabiyaka.net/chicha-whois/latest/no-gui/mac/amd64/chicha-whois -o /usr/local/bin/chicha-whois && sudo chmod +x /usr/local/bin/chicha-whois; /usr/local/bin/chicha-whois --version;
```

On macOS (Apple Silicon/ARM64):
```bash
sudo curl -L https://files.zabiyaka.net/chicha-whois/latest/no-gui/mac/arm64/chicha-whois -o /usr/local/bin/chicha-whois && sudo chmod +x /usr/local/bin/chicha-whois; /usr/local/bin/chicha-whois --version;
```

Done? Test it:
```bash
chicha-whois -h
```

---

## Commands

| Option                                  | Description                                                                                                                                                                 |
|-----------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `-u`                                    | Update / download the RIPE NCC database locally.                                                                                                                            |
| `-dns-acl COUNTRYCODE`                  | Generate a BIND ACL containing all IP ranges for `COUNTRYCODE` (e.g., `RU`). The result is saved as `acl_<COUNTRYCODE>.conf` in your home directory.                         |
| `-dns-acl-f COUNTRYCODE`                | Same as above, but filters out nested subnets for a smaller, optimized ACL.                                                                                                  |
| `-ovpn COUNTRYCODE`                     | Generate an OpenVPN exclude-route list with `route <ip> <mask> net_gateway`. Saved as `openvpn_exclude_<COUNTRYCODE>.txt`.                                                  |
| `-ovpn-f COUNTRYCODE`                   | Same as above, but filters out nested subnets (more efficient routes).                                                                                                       |
| `-l`                                    | List all available country codes recognized in the RIPE region.                                                                                                              |
| `-h`, `--help`                          | Display help.                                                                                                                                                                |
| `-v`, `--version`                       | Show application version.                                                                                                                                                    |
| **`-search [-dns | -ovpn | -ovpn-push] CC:kw1,kw2...`** | **Advanced search.** Allows searching by optional country code `CC` plus **one or more keywords** (`kw1`, `kw2`, ...). Filters out nested subnets, and **prints** the results to stdout in the chosen format (no file is created).  |

### About the `-search` Command

- Format:  
  ```
  chicha-whois -search [-dns | -ovpn | -ovpn-push] CC:kw1,kw2...
  ```
  
- **Country code** (`CC`) is optional.  
  - If you leave it empty (like `:kw1,kw2`), it will match blocks from **any** country if the keywords are found.  
  - If you specify a code (e.g., `RU`), it will only match blocks containing `country: RU`.
  
- **Keywords** are comma-separated. The tool searches for them in each RIPE DB block (case-insensitive). If a block contains at least one of these keywords, it is considered a match.

- **Sub-flags** for `-search`:
  - **`-dns`**: Print results in a DNS BIND ACL style (e.g. `acl "RU" { 1.2.3.0/24; };`).
  - **`-ovpn`**: Print results in an OpenVPN client style (`route <ip> <mask> net_gateway`).
  - **`-ovpn-push`**: Print results in an OpenVPN server push style (`push "route <ip> <mask>"`).

- **Example**:
  ```bash
  # Search for RU blocks containing any of these keywords: ok.ru, vk.ru, drive2.ru
  # Print them in DNS ACL style to the console (not saved to file):
  chicha-whois -search -dns RU:ok.ru,vk.ru,drive2.ru
  ```
  
  ```bash
  # Search for blocks in any country (no CC) containing google.com, amazon, cloudflare
  # Print them as OpenVPN server "push" routes in the console:
  chicha-whois -search -ovpn-push :google.com,amazon,cloudflare
  ```
  
- **Nested subnet filtering** is always applied when using `-search`, so you get the most optimized list of CIDRs without overlaps.

---

## Examples

1. **Update the RIPE database**  
   ```bash
   chicha-whois -u
   ```
   Downloads the latest `ripe.db.inetnum.gz` and decompresses it locally.

2. **Create a DNS ACL for Russia**  
   ```bash
   chicha-whois -dns-acl RU
   ```
   Generates `acl_RU.conf` in your home directory.

3. **Optimized DNS ACL for Russia**  
   ```bash
   chicha-whois -dns-acl-f RU
   ```
   Same as above but filters out redundant subnets.

4. **List all country codes**  
   ```bash
   chicha-whois -l
   ```
   Example result: RU - Russia, UA - Ukraine, etc.

5. **Generate an OpenVPN Exclusion List**  
   ```bash
   chicha-whois -ovpn RU
   ```
   Creates `openvpn_exclude_RU.txt` with route statements (unfiltered).

6. **Filtered OpenVPN Exclusion List**  
   ```bash
   chicha-whois -ovpn-f RU
   ```
   Same as above, but with CIDR aggregation to remove nested subnets.

7. **Search by country code and keywords**  
   ```bash
   chicha-whois -search -dns UA:google.com,kyivstar,mts
   ```
   - Finds all UA (`country: UA`) blocks that contain **any** of these keywords (`google.com`, `kyivstar`, `mts`) in the RIPE data.  
   - Removes duplicates and nested subnets.  
   - Prints them in a DNS ACL style block directly to the console.

8. **Search with no country code**  
   ```bash
   chicha-whois -search -ovpn-push :cloudflare,amazon
   ```
   - Matches **any** country if the block has at least one keyword.  
   - Prints them in server-style push format (for inclusion in an OpenVPN server config).

---

## Notes

- **Local DB Path**: `~/.ripe.db.cache/ripe.db.inetnum`  
- **DNS ACL Output**: `~/acl_<COUNTRYCODE>.conf` (for `-dns-acl` / `-dns-acl-f`).  
- **OpenVPN Output**: `~/openvpn_exclude_<COUNTRYCODE>.txt` (for `-ovpn` / `-ovpn-f`).  
- **`-search`** does **not** save output to file; everything is printed to stdout.  

---

## BIND9 Configuration Example (for RU & UA)

You can include these files (generated via `-dns-acl`) in your `named.conf`:

```
include "/etc/bind/acl_RU.conf";
include "/etc/bind/acl_UA.conf";

view "Russia" {
    match-clients { RU; };  # RU clients
    zone "domain.com" {
        type master;
        file "/etc/bind/zones/db.domain.com.RU";
    };
};

view "Ukraine" {
    match-clients { UA; };  # UA clients
    zone "domain.com" {
        type master;
        file "/etc/bind/zones/db.domain.com.UA";
    };
};

view "default" {
    match-clients { any; };  # All other clients
    zone "domain.com" {
        type master;
        file "/etc/bind/zones/db.domain.com.default";
    };
};
```

- Save ACLs to `/etc/bind/acl_RU.conf` and `/etc/bind/acl_UA.conf`.
- Create corresponding zone files, e.g.:
  ```
  /etc/bind/zones/db.domain.com.RU
  /etc/bind/zones/db.domain.com.UA
  /etc/bind/zones/db.domain.com.default
  ```
- Check config with `sudo named-checkconf`, then reload with `sudo systemctl restart bind9`.

---

## OpenVPN Exclusion Example

To exclude entire country routes from your VPN tunnel, append lines like these to your `.ovpn` client config:

```bash
# Redirect all traffic through VPN
redirect-gateway def1

# Exclude RU IPs from VPN
route 100.43.64.0 255.255.255.0 net_gateway
route 100.43.65.0 255.255.255.0 net_gateway
route 100.43.66.0 255.255.254.0 net_gateway
```

This ensures those specific subnets do **not** go through the VPN tunnel. If you want to **push** routes from the server side, you can use the `-ovpn-push` search flag to generate lines like `push "route <ip> <mask>"`.

---


Enjoy **chicha-whois**! If you have any questions or improvements, feel free to open an issue or a pull request.
