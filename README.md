# The RIPE Country IP List for DNS BIND Access Lists.

### What It Does

The program takes IP addresses and sorts them by country. Why? Because DNS BIND servers like order, and someone has to do it. So, it creates Access Control Lists (ACLs) for countries. Simple as that.

### How to Use It

- Run it.
- Tell it what you want (a country code, for example).
- Watch it craft a list for your BIND server.

#### Example:

```bash
./ripe-country-list -dns-acl US
```

This command creates an ACL for the United States. It might laugh, it might sigh, but it will deliver.

### Downloads

Pick your system and let the program do its magic.

- **Linux AMD64**: [Download](https://files.zabiyaka.net/ripe-country-list/latest/no-gui/linux/amd64/ripe-country-list)
- **Windows AMD64**: [Download](https://files.zabiyaka.net/ripe-country-list/latest/no-gui/windows/amd64/ripe-country-list.exe)
- **macOS AMD64**: [Download](https://files.zabiyaka.net/ripe-country-list/latest/no-gui/mac/amd64/ripe-country-list)
- **Linux ARM64**: [Download](https://files.zabiyaka.net/ripe-country-list/latest/no-gui/linux/arm64/ripe-country-list)

For other systems, explore [all binaries](https://files.zabiyaka.net/ripe-country-list/latest/no-gui/).

---

Run it. Let it list. Bind your DNS. And maybe, smile a little.
