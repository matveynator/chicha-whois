# chicha-whois

chicha-whois is a command-line tool to manage RIPE database and generate DNS ACLs.


## Installation Linux AMD64 systems:
For Linux AMD64 systems you can use the following command to download the chicha-whois binary, move it to /usr/local/bin, and make it executable:

```bash
sudo curl -L https://files.zabiyaka.net/chicha-whois/latest/no-gui/linux/amd64/chicha-whois -o /usr/local/bin/chicha-whois && sudo chmod +x /usr/local/bin/chicha-whois
```

Run `chicha-whois -h` to start.

## Commands

- `-u`: Update the RIPE database.
- `-dns-acl COUNTRYCODE`: Create a BIND ACL for a country (e.g., `RU`).
- `-dns-acl-f COUNTRYCODE`: Create a filtered BIND ACL (removes redundant subnets).
- `-l`: Show all country codes.
- `-h`: Show help.

## Examples

1. **Update the database:**
   ```bash
   chicha-whois -u
   ```
   Downloads and saves the RIPE database to `~/.ripe.db.cache/ripe.db.inetnum`.

2. **Generate a BIND ACL:**
   ```bash
   chicha-whois -dns-acl RU
   ```
   Creates `acl_RU.conf` in your home directory.

3. **Generate a filtered BIND ACL:**
   ```bash
   chicha-whois -dns-acl-f RU
   ```

4. **List available country codes:**
   ```bash
   chicha-whois -l
   ```

---

### Notes
- **Database location:** `~/.ripe.db.cache/ripe.db.inetnum`.
- **ACL files saved to:** Your home directory.

### Downloads

Pick your system and let the program do its magic.

- **Linux AMD64**: [Download](https://files.zabiyaka.net/chicha-whois/latest/no-gui/linux/amd64/chicha-whois)
- **Windows AMD64**: [Download](https://files.zabiyaka.net/chicha-whois/latest/no-gui/windows/amd64/chicha-whois.exe)
- **macOS AMD64**: [Download](https://files.zabiyaka.net/chicha-whois/latest/no-gui/mac/amd64/chicha-whois)
- **Linux ARM64**: [Download](https://files.zabiyaka.net/chicha-whois/latest/no-gui/linux/arm64/chicha-whois)

For other systems, explore [all binaries](https://files.zabiyaka.net/chicha-whois/latest/no-gui/).

