# Chicha-Whois

Chicha-Whois is a command-line tool designed to interact with the RIPE database. It helps system administrators manage IP ranges associated with specific countries by updating the database, generating ACLs for BIND DNS servers, and displaying available country codes.

## Usage

Run the `chicha-whois` command followed by an option:

```bash
chicha-whois <option>
```

### Options

- `-h`, `--help`: Show the help message.
- `-v`, `--version`: Display the application version.
- `-u`: Update the RIPE database.
- `-dns-acl COUNTRYCODE`: Generate an ACL list for BIND DNS based on the specified country code.
- `-dns-acl-f COUNTRYCODE`: Generate a filtered ACL list for BIND DNS, optimizing by removing redundant subnets.
- `-l`: Show available country codes.

### Examples

#### Update the RIPE Database

Before generating ACLs, update the RIPE database:

```bash
chicha-whois -u
```

This command:

- Downloads the latest RIPE database.
- Displays detailed progress during download and extraction.
- Saves the updated database to `~/.ripe.db.cache/ripe.db.inetnum`.

#### Generate a BIND ACL for a Country

To create an ACL file for a specific country (e.g., Russia):

```bash
chicha-whois -dns-acl RU
```

This command:

- Checks for the RIPE database and prompts you to update if it's missing.
- Parses the database to find all IP ranges associated with the country code `RU`.
- Generates an ACL file named `acl_RU.conf` in your home directory.
- The ACL file contains all the IP ranges in CIDR notation for use with BIND DNS.

#### Generate a Filtered BIND ACL

For an optimized ACL by filtering out redundant subnets:

```bash
chicha-whois -dns-acl-f RU
```

This command performs the same steps as `-dns-acl` but additionally filters out any subnets that are subsets of larger networks, resulting in a more efficient ACL file with fewer entries.

#### List Available Country Codes

To display all supported country codes and their corresponding country names:

```bash
chicha-whois -l
```

This command:

- Prints a sorted list of country codes along with country names.
- Helps you find the correct country code to use with other commands.

## Notes

- **RIPE Database Location**: The RIPE database is cached at `~/.ripe.db.cache/ripe.db.inetnum`.
- **ACL File Location**: Generated ACL files are saved in your home directory (e.g., `~/acl_RU.conf`).
- **Internet Connection**: Required to update the RIPE database.
- **Permissions**: Ensure you have the necessary permissions to read/write in your home directory.

### Downloads

Pick your system and let the program do its magic.

- **Linux AMD64**: [Download](https://files.zabiyaka.net/chicha-whois/latest/no-gui/linux/amd64/chicha-whois)
- **Windows AMD64**: [Download](https://files.zabiyaka.net/chicha-whois/latest/no-gui/windows/amd64/chicha-whois.exe)
- **macOS AMD64**: [Download](https://files.zabiyaka.net/chicha-whois/latest/no-gui/mac/amd64/chicha-whois)
- **Linux ARM64**: [Download](https://files.zabiyaka.net/chicha-whois/latest/no-gui/linux/arm64/chicha-whois)

For other systems, explore [all binaries](https://files.zabiyaka.net/chicha-whois/latest/no-gui/).

