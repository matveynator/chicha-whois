<img src="https://raw.githubusercontent.com/matveynator/chicha-whois/refs/heads/master/chicha-whois-logo.png" alt="chicha-whois" width="50%" align="right" />


**chicha-whois** is a tiny but powerful CLI tool for working with the RIPE database and generating DNS ACLs. Clean, simple, and gets the job done. Written in Golang.



## Downloads

Pick your binary and get started:

- [Linux AMD64](https://files.zabiyaka.net/chicha-whois/latest/no-gui/linux/amd64/chicha-whois)  
- [Windows AMD64](https://files.zabiyaka.net/chicha-whois/latest/no-gui/windows/amd64/chicha-whois.exe)  
- [macOS AMD64](https://files.zabiyaka.net/chicha-whois/latest/no-gui/mac/amd64/chicha-whois)  
- [Linux ARM64](https://files.zabiyaka.net/chicha-whois/latest/no-gui/linux/arm64/chicha-whois)  

Need something else? [Check all binaries](https://files.zabiyaka.net/chicha-whois/latest/no-gui/).

---

## Installation

On Linux AMD64, install in one line:  

```bash
sudo curl -L https://files.zabiyaka.net/chicha-whois/latest/no-gui/linux/amd64/chicha-whois -o /usr/local/bin/chicha-whois && sudo chmod +x /usr/local/bin/chicha-whois
```

Done? Try it:  
```bash
chicha-whois -h
```

---

## Commands

- `-u`: Update the RIPE database.
- `-dns-acl COUNTRYCODE`: Generate a BIND ACL (e.g., `RU`).
- `-dns-acl-f COUNTRYCODE`: Create a filtered ACL (no redundant subnets).
- `-l`: Show available country codes.
- `-h`: Show help.

---

## Examples

1. **Update the RIPE database**  
   ```bash
   chicha-whois -u
   ```
   This downloads and updates the database locally.

2. **Create an ACL for Russia**  
   ```bash
   chicha-whois -dns-acl RU
   ```
   Outputs `acl_RU.conf` with all Russian IP ranges.

3. **Optimized ACL**  
   ```bash
   chicha-whois -dns-acl-f RU
   ```
   Same as above, but smarter—filters out redundant subnets.

4. **List all country codes**  
   ```bash
   chicha-whois -l
   ```

---

## Notes

- **Database saved to**: `~/.ripe.db.cache/ripe.db.inetnum`.  
- **ACL files saved to**: Your home directory (e.g., `~/acl_RU.conf`).  


## BIND9 Configuration for RU and UA Clients
Copy and paste the following configuration into your BIND9 named.conf:

```
include "/etc/bind/acl_RU.conf";
include "/etc/bind/acl_UA.conf";

view "RU" {
    match-clients { RU; };  # RU clients
    zone "domain.com" {
        type master;
        file "/etc/bind/zones/db.domain.com.RU";
    };
};

view "UA" {
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

Save ACLs to /etc/bind/acl_RU.conf and /etc/bind/acl_UA.conf.

### Create zone files:

```
/etc/bind/zones/db.domain.com.RU
/etc/bind/zones/db.domain.com.UA
/etc/bind/zones/db.domain.com.default
```

### Verify configuration:

```
sudo named-checkconf
sudo named-checkzone domain.com /etc/bind/zones/db.domain.com.ru
```

### Restart BIND:
```
sudo systemctl restart bind9
```
Done!

