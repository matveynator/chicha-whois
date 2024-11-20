<img src="https://repository-images.githubusercontent.com/890922366/c688698f-4bb2-4aad-a6ed-372983654b34" width="50%" align="right">

# chicha-whois

**chicha-whois** is a tiny but powerful CLI tool for working with the RIPE database and generating DNS ACLs. Clean, simple, and gets the job done.

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
