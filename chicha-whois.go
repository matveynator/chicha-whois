package main

import (
    "bufio"
    "bytes"
    "compress/gzip"
    "encoding/binary"
    "fmt"
    "io"
    "math/bits"
    "net"
    "net/http"
    "os"
    "path/filepath"
    "sort"
    "strings"
)

// version    - The current application version. Set to "dev" by default.
// ripedbPath - The file path to the cached RIPE DB file (determined at runtime).
var (
    version    = "dev"
    ripedbPath string
)

// ProgressReader is a wrapper around an io.Reader that displays progress while reading bytes.
type ProgressReader struct {
    Reader    io.Reader // Underlying reader (for example, the HTTP response body).
    Total     int64     // Total size of the data to read (for showing progress percentage).
    Progress  int64     // Number of bytes read so far.
    Operation string    // Description of the current operation, e.g., "Downloading".
}

// Read updates ProgressReader's Progress count and prints progress information.
func (pr *ProgressReader) Read(p []byte) (int, error) {
    n, err := pr.Reader.Read(p)
    pr.Progress += int64(n)

    if pr.Total > 0 {
        percent := float64(pr.Progress) / float64(pr.Total) * 100
        fmt.Printf("\r%s... %.2f%%", pr.Operation, percent)
    } else {
        fmt.Printf("\r%s... %d bytes", pr.Operation, pr.Progress)
    }
    return n, err
}

func main() {
    // Attempt to determine the current user's home directory.
    homeDir, err := os.UserHomeDir()
    if err != nil {
        fmt.Println("Error getting home directory:", err)
        return
    }

    // Build the default path to the RIPE DB cache file.
    ripedbPath = filepath.Join(homeDir, ".ripe.db.cache/ripe.db.inetnum")

    // Check if any arguments were provided.
    if len(os.Args) < 2 {
        usage()
        return
    }

    // The first argument is the command.
    cmd := os.Args[1]

    switch cmd {
    case "-h", "--help":
        // Print usage message.
        usage()

    case "-l":
        // Print known country codes (and their full names).
        showAvailableCountryCodes()

    case "-v", "--version":
        // Print application version.
        fmt.Printf("version: %s\n", version)

    case "-u":
        // Update / download and decompress the RIPE database into the local cache.
        updateRIPEdb()

    //--------------------------------------------------------------------
    // Old flags that write output to files (left unchanged)
    //--------------------------------------------------------------------
    case "-dns-acl":
        // Generate an unfiltered BIND ACL file for the provided country code.
        if len(os.Args) > 2 {
            countryCode := os.Args[2]
            ensureRIPEdb()
            createBindACL(countryCode)
        } else {
            usage()
        }

    case "-dns-acl-f":
        // Generate a filtered BIND ACL (remove nested subnets) for the provided country code.
        if len(os.Args) > 2 {
            countryCode := os.Args[2]
            ensureRIPEdb()
            createBindACLFiltered(countryCode)
        } else {
            usage()
        }

    case "-ovpn":
        // Generate an unfiltered OpenVPN route list for the given country code.
        if len(os.Args) > 2 {
            countryCode := os.Args[2]
            ensureRIPEdb()
            createOpenVPNExclude(countryCode)
        } else {
            usage()
        }

    case "-ovpn-f":
        // Generate a filtered OpenVPN route list (remove nested subnets) for the given country code.
        if len(os.Args) > 2 {
            countryCode := os.Args[2]
            ensureRIPEdb()
            createOpenVPNExcludeFiltered(countryCode)
        } else {
            usage()
        }

    //--------------------------------------------------------------------
    // New -search flag: search by country code (optional) + keywords,
    // filter nested subnets, and print to the screen in various formats
    //--------------------------------------------------------------------
    case "-search":
        if len(os.Args) < 3 {
            usage()
            return
        }

        // Make sure the RIPE DB file is available.
        ensureRIPEdb()

        // Default output mode: just print the found ranges in plain text.
        outputMode := "print"

        // We look for optional sub-flags: -dns, -ovpn, or -ovpn-push.
        // Once we find something that doesn't match those sub-flags,
        // we assume it's the actual search parameter (CC:keywords).
        var searchIndex int
        for i := 2; i < len(os.Args); i++ {
            arg := os.Args[i]
            switch arg {
            case "-dns":
                outputMode = "dns"
            case "-ovpn":
                outputMode = "ovpn"
            case "-ovpn-push":
                outputMode = "ovpn-push"
            default:
                // This must be the search parameter (e.g. "RU:ok.ru,vk.ru")
                searchIndex = i
                break
            }
            if searchIndex != 0 {
                break
            }
        }

        // If we never found the search parameter, show usage and exit.
        if searchIndex == 0 {
            usage()
            return
        }

        // Parse the search parameter "CC:kw1,kw2,kw3..."
        searchParam := os.Args[searchIndex]
        var countryCode string
        var keywords []string

        parts := strings.SplitN(searchParam, ":", 2)
        if len(parts) == 2 {
            // Everything before ':' is the country code (could be empty),
            // everything after ':' is a comma-separated list of keywords.
            countryCode = strings.TrimSpace(parts[0])
            kwStr := strings.TrimSpace(parts[1])
            if kwStr != "" {
                keywords = strings.Split(kwStr, ",")
            }
        } else {
            // If no colon is present, treat the entire string as a country code,
            // and there are no keywords.
            countryCode = searchParam
        }

        // Trim whitespace in the keywords.
        for i := range keywords {
            keywords[i] = strings.TrimSpace(keywords[i])
        }

        fmt.Printf("Performing a RIPE database search:\n  Country code: '%s', Keywords: %v\n",
            countryCode, keywords)

        // Extract matching CIDRs.
        ipRanges := extractCIDRsByKeywordsAndCountry(countryCode, keywords, ripedbPath, false)
        if len(ipRanges) == 0 {
            fmt.Println("Nothing found for the specified criteria.")
            return
        }

        // Remove duplicates.
        ipRanges = removeDuplicates(ipRanges)
        // Filter out nested subnets (always).
        ipRanges = filterRedundantCIDRs(ipRanges)
        // Sort them in ascending order.
        sort.Strings(ipRanges)

        // Print to the console based on the chosen format.
        switch outputMode {
        case "dns":
            // DNS BIND ACL format, but print to the console instead of writing a file.
            aclName := countryCode
            if aclName == "" {
                aclName = "search"
            }
            fmt.Printf("\nacl \"%s\" {\n", aclName)
            for _, cidr := range ipRanges {
                fmt.Printf("  %s;\n", cidr)
            }
            fmt.Println("};")

        case "ovpn":
            // OpenVPN client-style format (using net_gateway).
            cc := countryCode
            if cc == "" {
                cc = "SEARCH"
            }
            fmt.Println("# Redirect all traffic through VPN")
            fmt.Println("redirect-gateway def1")
            fmt.Println()
            fmt.Printf("# Exclude %s IP ranges from the VPN\n", strings.ToUpper(cc))

            for _, cidr := range ipRanges {
                startIP, netmask, err := cidrToRoute(cidr)
                if err != nil {
                    continue
                }
                line := fmt.Sprintf("route %s %s net_gateway", startIP, netmask)
                fmt.Println(line)
            }

        case "ovpn-push":
            // OpenVPN server-style format (push directives).
            cc := countryCode
            if cc == "" {
                cc = "SEARCH"
            }
            fmt.Println("# Redirect all traffic through VPN (server pushes these directives)")
            fmt.Println("push \"redirect-gateway def1\"")
            fmt.Println()
            fmt.Printf("# Exclude %s IP ranges from the VPN (pushed to clients)\n", strings.ToUpper(cc))

            for _, cidr := range ipRanges {
                startIP, netmask, err := cidrToRoute(cidr)
                if err != nil {
                    continue
                }
                line := fmt.Sprintf("push \"route %s %s net_gateway\"", startIP, netmask)
                fmt.Println(line)
            }

        default:
            // If no format specified, just print the final CIDR list.
            fmt.Println("Found CIDR ranges (after filtering):")
            for _, cidr := range ipRanges {
                fmt.Println(" ", cidr)
            }
        }

    default:
        usage()
    }
}

// usage prints a help message describing all command-line options.
func usage() {
    fmt.Println(`Usage: chicha-whois <option>

Options:
  -h, --help               Show this help message
  -v, --version            Show application version
  -u                       Update local RIPE NCC database cache
  -l                       List available country codes

  # Generate DNS Bind ACL (unfiltered / filtered) [writes output to a file]
  -dns-acl COUNTRYCODE     Generate unfiltered DNS ACL file for BIND
  -dns-acl-f COUNTRYCODE   Generate filtered DNS ACL file for BIND (removes nested subnets)

  # Generate OpenVPN exclude-route list (unfiltered / filtered) [writes output to a file]
  -ovpn COUNTRYCODE        Generate unfiltered OpenVPN routes
  -ovpn-f COUNTRYCODE      Generate filtered OpenVPN routes (removes nested subnets)

  # New: Search by country code (optional) AND/OR keywords, filter subnets, print results to screen
  # Syntax:
  #   chicha-whois -search [-dns | -ovpn | -ovpn-push] CC:kw1,kw2,...
  #
  # Examples:
  #   chicha-whois -search -dns RU:ok.ru,vkontakte,mts,megafon.ru
  #   chicha-whois -search -ovpn-push :google.com,cloudflare,amazon
  #   chicha-whois -search -ovpn UA:gmail,outlook`)
}

// ensureRIPEdb checks whether the RIPE DB cache file exists; if not, triggers an update.
func ensureRIPEdb() {
    if _, err := os.Stat(ripedbPath); os.IsNotExist(err) {
        fmt.Println("RIPE database cache not found. Attempting to update...")
        updateRIPEdb()
    }
}

// updateRIPEdb downloads the RIPE database from a public URL, then decompresses it.
func updateRIPEdb() {
    downloadURL := "https://ftp.ripe.net/ripe/dbase/split/ripe.db.inetnum.gz"

    homeDir, err := os.UserHomeDir()
    if err != nil {
        fmt.Println("Error getting home directory:", err)
        return
    }

    // Create a temporary file for the gzip data.
    tmpFile, err := os.CreateTemp(homeDir, "ripe.db.inetnum-*.gz")
    if err != nil {
        fmt.Println("Error creating temporary file:", err)
        return
    }
    defer func() {
        _ = os.Remove(tmpFile.Name())
        fmt.Println("Temporary file removed:", tmpFile.Name())
    }()
    defer tmpFile.Close()

    fmt.Printf("Starting download of the RIPE database from %s\n", downloadURL)
    fmt.Printf("Saving to temporary file: %s\n", tmpFile.Name())

    resp, err := http.Get(downloadURL)
    if err != nil {
        fmt.Printf("Error downloading RIPE database: %v\n", err)
        return
    }
    defer resp.Body.Close()

    totalSize := resp.ContentLength
    if totalSize <= 0 {
        fmt.Println("Warning: unable to determine file size for progress display.")
    } else {
        fmt.Printf("Total file size: %d bytes\n", totalSize)
    }

    progressReader := &ProgressReader{
        Reader:    resp.Body,
        Total:     totalSize,
        Operation: "Downloading",
    }

    // Copy the downloaded bytes to the temporary file, showing progress.
    _, err = io.Copy(tmpFile, progressReader)
    if err != nil {
        fmt.Println("Error writing to temporary file:", err)
        return
    }
    fmt.Println() // New line after final progress output.

    // Now decompress the downloaded .gz into ripedbPath.
    fmt.Printf("Extracting %s to %s\n", tmpFile.Name(), ripedbPath)
    if err := gunzipFileWithProgress(tmpFile.Name(), ripedbPath); err != nil {
        fmt.Println("Error decompressing RIPE database:", err)
        return
    }

    fmt.Printf("RIPE database updated successfully at %s\n", ripedbPath)
}

// gunzipFileWithProgress decompresses a .gz file and writes the output to a destination file.
func gunzipFileWithProgress(source, destination string) error {
    fi, err := os.Stat(source)
    if err != nil {
        return err
    }
    compressedSize := fi.Size()

    // Ensure the destination directory exists, create if needed.
    dir := filepath.Dir(destination)
    if err = os.MkdirAll(dir, os.ModePerm); err != nil {
        return fmt.Errorf("error creating directory %s: %v", dir, err)
    }

    file, err := os.Open(source)
    if err != nil {
        return err
    }
    defer file.Close()

    progressReader := &ProgressReader{
        Reader: file,
        Total:  compressedSize,
    }

    gz, err := gzip.NewReader(progressReader)
    if err != nil {
        return err
    }
    defer gz.Close()

    out, err := os.Create(destination)
    if err != nil {
        return err
    }
    defer out.Close()

    _, err = io.Copy(out, gz)
    if err != nil {
        return err
    }
    fmt.Println("\nDecompression completed.")
    return nil
}

//-------------------------------------------------------------------------
// Functions that write DNS/OVPN output to files (for older flags)
//-------------------------------------------------------------------------

// createBindACL creates an unfiltered DNS BIND ACL file for the specified country code.
func createBindACL(countryCode string) {
    fmt.Printf("Creating BIND ACL file for country code: %s\n", countryCode)

    ipRanges := extractCountryCIDRs(countryCode, ripedbPath, false)
    if len(ipRanges) == 0 {
        fmt.Printf("No IP ranges found for country code: %s\n", countryCode)
        return
    }

    ipRanges = removeDuplicates(ipRanges)
    sort.Strings(ipRanges)

    homeDir, _ := os.UserHomeDir()
    aclFilePath := filepath.Join(homeDir, fmt.Sprintf("acl_%s.conf", countryCode))

    var entries []string
    for _, cidr := range ipRanges {
        entries = append(entries, fmt.Sprintf("  %s;", cidr))
    }
    aclContent := fmt.Sprintf("acl \"%s\" {\n%s\n};\n", countryCode, strings.Join(entries, "\n"))

    if err := os.WriteFile(aclFilePath, []byte(aclContent), 0644); err != nil {
        fmt.Printf("Error writing BIND ACL file: %v\n", err)
        return
    }
    fmt.Printf("BIND ACL file created at: %s\n", aclFilePath)
}

// createBindACLFiltered creates a DNS BIND ACL file after removing nested subnets.
func createBindACLFiltered(countryCode string) {
    fmt.Printf("Creating BIND ACL file (filtered) for country code: %s\n", countryCode)

    ipRanges := extractCountryCIDRs(countryCode, ripedbPath, false)
    if len(ipRanges) == 0 {
        fmt.Printf("No IP ranges found for country code: %s\n", countryCode)
        return
    }

    ipRanges = removeDuplicates(ipRanges)
    ipRanges = filterRedundantCIDRs(ipRanges)
    sort.Strings(ipRanges)

    homeDir, _ := os.UserHomeDir()
    aclFilePath := filepath.Join(homeDir, fmt.Sprintf("acl_%s.conf", countryCode))

    var entries []string
    for _, cidr := range ipRanges {
        entries = append(entries, fmt.Sprintf("  %s;", cidr))
    }
    aclContent := fmt.Sprintf("acl \"%s\" {\n%s\n};\n", countryCode, strings.Join(entries, "\n"))

    if err := os.WriteFile(aclFilePath, []byte(aclContent), 0644); err != nil {
        fmt.Printf("Error writing filtered BIND ACL file: %v\n", err)
        return
    }
    fmt.Printf("Filtered BIND ACL file created at: %s\n", aclFilePath)
}

// createOpenVPNExclude creates an unfiltered OpenVPN exclude-route file for the given country code.
func createOpenVPNExclude(countryCode string) {
    fmt.Printf("Creating an unfiltered OpenVPN exclude-route file for country code: %s\n", countryCode)

    ipRanges := extractCountryCIDRs(countryCode, ripedbPath, false)
    if len(ipRanges) == 0 {
        fmt.Printf("No IP ranges found for country code: %s\n", countryCode)
        return
    }

    ipRanges = removeDuplicates(ipRanges)
    sort.Strings(ipRanges)

    var routeLines []string
    routeLines = append(routeLines,
        "# Redirect all traffic through VPN",
        "push \"redirect-gateway def1\"",
        "",
        fmt.Sprintf("# Exclude %s IPs from VPN", strings.ToUpper(countryCode)),
    )

    for _, cidr := range ipRanges {
        startIP, netmask, err := cidrToRoute(cidr)
        if err != nil {
            fmt.Printf("Skipping CIDR (%s): %v\n", cidr, err)
            continue
        }
        line := fmt.Sprintf("push \"route %s %s net_gateway\"", startIP, netmask)
        routeLines = append(routeLines, line)
    }

    homeDir, _ := os.UserHomeDir()
    outFilePath := filepath.Join(homeDir, fmt.Sprintf("openvpn_exclude_%s.txt", strings.ToUpper(countryCode)))

    content := strings.Join(routeLines, "\n") + "\n"
    if err := os.WriteFile(outFilePath, []byte(content), 0644); err != nil {
        fmt.Printf("Error writing OpenVPN exclude file: %v\n", err)
        return
    }
    fmt.Printf("OpenVPN exclude-route file created at: %s\n", outFilePath)
}

// createOpenVPNExcludeFiltered creates a filtered OpenVPN exclude-route file for the given country code.
func createOpenVPNExcludeFiltered(countryCode string) {
    fmt.Printf("Creating a filtered OpenVPN exclude-route file for country code: %s\n", countryCode)

    ipRanges := extractCountryCIDRs(countryCode, ripedbPath, false)
    if len(ipRanges) == 0 {
        fmt.Printf("No IP ranges found for country code: %s\n", countryCode)
        return
    }

    ipRanges = removeDuplicates(ipRanges)
    ipRanges = filterRedundantCIDRs(ipRanges)
    sort.Strings(ipRanges)

    var routeLines []string
    routeLines = append(routeLines,
        "# Redirect all traffic through VPN",
        "push \"redirect-gateway def1\"",
        "",
        fmt.Sprintf("# Exclude %s IPs from VPN (filtered)", strings.ToUpper(countryCode)),
    )

    for _, cidr := range ipRanges {
        startIP, netmask, err := cidrToRoute(cidr)
        if err != nil {
            fmt.Printf("Skipping CIDR (%s): %v\n", cidr, err)
            continue
        }
        line := fmt.Sprintf("push \"route %s %s net_gateway\"", startIP, netmask)
        routeLines = append(routeLines, line)
    }

    homeDir, _ := os.UserHomeDir()
    outFilePath := filepath.Join(homeDir, fmt.Sprintf("openvpn_exclude_%s.txt", strings.ToUpper(countryCode)))

    content := strings.Join(routeLines, "\n") + "\n"
    if err := os.WriteFile(outFilePath, []byte(content), 0644); err != nil {
        fmt.Printf("Error writing filtered OpenVPN exclude file: %v\n", err)
        return
    }
    fmt.Printf("Filtered OpenVPN exclude-route file created at: %s\n", outFilePath)
}

//-------------------------------------------------------------------------
// Parsing CIDRs and converting net.IPMask to dotted notation
//-------------------------------------------------------------------------

// cidrToRoute parses a string like "192.168.0.0/24" into "192.168.0.0" and "255.255.255.0".
func cidrToRoute(cidr string) (string, string, error) {
    _, ipNet, err := net.ParseCIDR(cidr)
    if err != nil {
        return "", "", fmt.Errorf("failed to parse CIDR: %v", err)
    }
    networkAddr := ipNet.IP.To4()
    if networkAddr == nil {
        return "", "", fmt.Errorf("IPv6 addresses are not supported in this example")
    }
    netmask := ipMaskToDotted(ipNet.Mask)
    return networkAddr.String(), netmask, nil
}

// ipMaskToDotted converts a net.IPMask (e.g., /24) to its dotted-decimal string (e.g., "255.255.255.0").
func ipMaskToDotted(mask net.IPMask) string {
    if len(mask) != 4 {
        return ""
    }
    return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}

//-------------------------------------------------------------------------
// Searching by country code or keywords
//-------------------------------------------------------------------------

// extractCountryCIDRs returns a list of CIDRs for inetnum blocks that match the given country code exactly.
func extractCountryCIDRs(countryCode, dbPath string, debugPrint bool) []string {
    file, err := os.Open(dbPath)
    if err != nil {
        fmt.Println("Error opening the RIPE database:", err)
        return nil
    }
    defer file.Close()

    countryCode = strings.ToUpper(countryCode)
    scanner := bufio.NewScanner(file)
    var ipRanges []string
    var blockLines []string

    for {
        blockLines = nil
        for scanner.Scan() {
            line := scanner.Text()
            if line == "" {
                break
            }
            blockLines = append(blockLines, line)
        }
        if len(blockLines) == 0 {
            // End of file
            break
        }

        var inetnumLine, countryLine string
        for _, line := range blockLines {
            line = strings.TrimSpace(line)
            if strings.HasPrefix(line, "inetnum:") {
                inetnumLine = line
            } else if strings.HasPrefix(line, "country:") {
                countryLine = line
            }
        }

        if inetnumLine != "" && countryLine != "" {
            fields := strings.Fields(countryLine)
            if len(fields) >= 2 {
                blockCountryCode := strings.ToUpper(fields[1])
                if blockCountryCode == countryCode {
                    // This block matches the specified country code.
                    inetnumFields := strings.Fields(inetnumLine)
                    if len(inetnumFields) >= 2 {
                        ipRangeStr := strings.Join(inetnumFields[1:], " ")
                        ipRangeParts := strings.Split(ipRangeStr, "-")
                        if len(ipRangeParts) == 2 {
                            start := strings.TrimSpace(ipRangeParts[0])
                            end := strings.TrimSpace(ipRangeParts[1])

                            if debugPrint {
                                fmt.Printf("Found inetnum entry: %s - %s\n", start, end)
                            }

                            cidr := generateCIDR(start, end)
                            if cidr != "" {
                                if debugPrint {
                                    fmt.Printf("Converted to CIDR: %s\n", cidr)
                                }
                                ipRanges = append(ipRanges, cidr)
                            }
                        }
                    }
                }
            }
        }
    }
    return ipRanges
}

// extractCIDRsByKeywordsAndCountry searches the RIPE DB for inetnum blocks that optionally match a country code
// and contain at least one of the provided keywords. 
func extractCIDRsByKeywordsAndCountry(countryCode string, keywords []string, dbPath string, debugPrint bool) []string {
    file, err := os.Open(dbPath)
    if err != nil {
        fmt.Println("Error opening the RIPE database:", err)
        return nil
    }
    defer file.Close()

    // Convert country code to uppercase for matching "country: XX".
    countryCode = strings.ToUpper(countryCode)

    // Convert all keywords to lowercase for case-insensitive search.
    for i := range keywords {
        keywords[i] = strings.ToLower(keywords[i])
    }

    scanner := bufio.NewScanner(file)
    var ipRanges []string
    var blockLines []string

    for {
        blockLines = nil
        for scanner.Scan() {
            line := scanner.Text()
            if line == "" {
                // Blank line signals the end of a block
                break
            }
            blockLines = append(blockLines, line)
        }
        if len(blockLines) == 0 {
            // End of file
            break
        }

        var inetnumLine, countryLine string
        for _, line := range blockLines {
            trimLine := strings.TrimSpace(line)
            if strings.HasPrefix(trimLine, "inetnum:") {
                inetnumLine = trimLine
            } else if strings.HasPrefix(trimLine, "country:") {
                countryLine = trimLine
            }
        }

        // If a country code was specified, check if the block matches it.
        if countryCode != "" {
            if countryLine == "" {
                // No country line in this block, skip it.
                continue
            }
            fields := strings.Fields(countryLine)
            if len(fields) < 2 || strings.ToUpper(fields[1]) != countryCode {
                // The country code in this block doesn't match the desired one.
                continue
            }
        }

        // If no keywords were given, we accept the block if it has an inetnum line.
        if len(keywords) == 0 {
            if inetnumLine != "" {
                ipRanges = append(ipRanges, inetnumToCIDR(inetnumLine, debugPrint)...)
            }
            continue
        }

        // Otherwise, we check if the block contains any of the keywords (case-insensitive).
        blockTextLower := strings.ToLower(strings.Join(blockLines, "\n"))
        match := false
        for _, kw := range keywords {
            if kw == "" {
                continue
            }
            if strings.Contains(blockTextLower, kw) {
                match = true
                break
            }
        }
        if match && inetnumLine != "" {
            ipRanges = append(ipRanges, inetnumToCIDR(inetnumLine, debugPrint)...)
        }
    }

    return ipRanges
}

// inetnumToCIDR parses a line like "inetnum: 1.2.3.0 - 1.2.3.255" and converts it to a CIDR range if possible.
func inetnumToCIDR(inetnumLine string, debugPrint bool) []string {
    var result []string
    parts := strings.Fields(inetnumLine)
    if len(parts) < 2 {
        return result
    }

    ipRangeStr := strings.Join(parts[1:], " ")
    ipRangeParts := strings.Split(ipRangeStr, "-")
    if len(ipRangeParts) == 2 {
        start := strings.TrimSpace(ipRangeParts[0])
        end := strings.TrimSpace(ipRangeParts[1])

        if debugPrint {
            fmt.Printf("Found inetnum entry: %s - %s\n", start, end)
        }

        cidr := generateCIDR(start, end)
        if cidr != "" {
            if debugPrint {
                fmt.Printf("Converted to CIDR: %s\n", cidr)
            }
            result = append(result, cidr)
        }
    }
    return result
}

// generateCIDR attempts to combine a start IP and end IP into a single CIDR notation (e.g., 192.168.0.0/24).
// This works properly if the range aligns to a power-of-two boundary.
func generateCIDR(startIPStr, endIPStr string) string {
    startIP := net.ParseIP(startIPStr).To4()
    endIP := net.ParseIP(endIPStr).To4()
    if startIP == nil || endIP == nil {
        fmt.Printf("Error: Invalid IP range: %s - %s\n", startIPStr, endIPStr)
        return ""
    }

    start := binary.BigEndian.Uint32(startIP)
    end := binary.BigEndian.Uint32(endIP)

    diff := start ^ end
    prefixLength := 32 - bits.Len32(diff)

    network := start &^ ((1 << (32 - prefixLength)) - 1)
    networkIP := make(net.IP, 4)
    binary.BigEndian.PutUint32(networkIP, network)

    return fmt.Sprintf("%s/%d", networkIP.String(), prefixLength)
}

//-------------------------------------------------------------------------
// Utility functions to filter out nested subnets, remove duplicates, etc.
//-------------------------------------------------------------------------

// filterRedundantCIDRs removes subnets that are fully contained inside larger subnets.
func filterRedundantCIDRs(cidrs []string) []string {
    var parsedCIDRs []*net.IPNet
    for _, cidrStr := range cidrs {
        _, ipNet, err := net.ParseCIDR(cidrStr)
        if err != nil {
            fmt.Printf("Error parsing CIDR %s: %v\n", cidrStr, err)
            continue
        }
        parsedCIDRs = append(parsedCIDRs, ipNet)
    }

    // Sort by prefix length ascending (bigger networks first), then by IP address ascending.
    sort.Slice(parsedCIDRs, func(i, j int) bool {
        onesI, bitsI := parsedCIDRs[i].Mask.Size()
        onesJ, bitsJ := parsedCIDRs[j].Mask.Size()

        // For IPv4, bitsI == 32; but let's keep this for correctness if needed.
        if bitsI != bitsJ {
            return bitsI < bitsJ
        }
        if onesI != onesJ {
            return onesI < onesJ
        }
        return bytes.Compare(parsedCIDRs[i].IP, parsedCIDRs[j].IP) < 0
    })

    var keptCIDRs []*net.IPNet
    for _, candidate := range parsedCIDRs {
        redundant := false
        for _, keeper := range keptCIDRs {
            if cidrContains(keeper, candidate) {
                redundant = true
                fmt.Printf("Filtered out redundant CIDR: %s (contained in %s)\n",
                    candidate.String(), keeper.String())
                break
            }
        }
        if !redundant {
            keptCIDRs = append(keptCIDRs, candidate)
        }
    }

    var results []string
    for _, net := range keptCIDRs {
        results = append(results, net.String())
    }
    return results
}

// cidrContains checks if 'inner' is fully contained within 'outer'.
func cidrContains(outer, inner *net.IPNet) bool {
    if !outer.Contains(inner.IP) {
        return false
    }
    innerLast := lastIP(inner)
    return outer.Contains(innerLast)
}

// lastIP calculates the broadcast (last) address in a subnet range.
func lastIP(ipNet *net.IPNet) net.IP {
    ip := ipNet.IP.To4()
    if ip == nil {
        // IPv6 is skipped in this example
        return nil
    }
    mask := ipNet.Mask
    network := ip.Mask(mask)
    broadcast := make(net.IP, len(network))

    for i := 0; i < len(network); i++ {
        broadcast[i] = network[i] | ^mask[i]
    }
    return broadcast
}

// removeDuplicates removes duplicate strings from a slice while preserving order.
func removeDuplicates(elements []string) []string {
    seen := make(map[string]bool)
    var result []string
    for _, e := range elements {
        if !seen[e] {
            seen[e] = true
            result = append(result, e)
        }
    }
    return result
}

//-------------------------------------------------------------------------
// List of available country codes
//-------------------------------------------------------------------------

// showAvailableCountryCodes prints a list of known country codes within the RIPE NCC region, sorted alphabetically by name.
func showAvailableCountryCodes() {
    countries := map[string]string{
        "AL": "Albania", "AM": "Armenia", "AT": "Austria", "AZ": "Azerbaijan",
        "BA": "Bosnia and Herzegovina", "BE": "Belgium", "BG": "Bulgaria",
        "BY": "Belarus", "CH": "Switzerland", "CY": "Cyprus", "CZ": "Czech Republic",
        "DE": "Germany", "DK": "Denmark", "EE": "Estonia", "ES": "Spain",
        "FI": "Finland", "FR": "France", "GE": "Georgia", "GR": "Greece",
        "HR": "Croatia", "HU": "Hungary", "IE": "Ireland", "IL": "Israel",
        "IS": "Iceland", "IT": "Italy", "KG": "Kyrgyzstan", "KZ": "Kazakhstan",
        "LT": "Lithuania", "LU": "Luxembourg", "LV": "Latvia", "MD": "Moldova",
        "ME": "Montenegro", "MK": "North Macedonia", "MT": "Malta", "NL": "Netherlands",
        "NO": "Norway", "PL": "Poland", "PT": "Portugal", "RO": "Romania",
        "RS": "Serbia", "RU": "Russia", "SE": "Sweden", "SI": "Slovenia",
        "SK": "Slovakia", "TJ": "Tajikistan", "TM": "Turkmenistan", "TR": "Turkey",
        "UA": "Ukraine", "UZ": "Uzbekistan",
    }

    var countryList []struct {
        Code string
        Name string
    }
    for code, name := range countries {
        countryList = append(countryList, struct {
            Code string
            Name string
        }{Code: code, Name: name})
    }

    // Sort by Name (alphabetically)
    sort.Slice(countryList, func(i, j int) bool {
        return countryList[i].Name < countryList[j].Name
    })

    fmt.Println("Available country codes and names (sorted by name):")
    for _, c := range countryList {
        fmt.Printf("%s - %s\n", c.Code, c.Name)
    }
}

