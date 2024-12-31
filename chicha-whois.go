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

// version    - holds the current application version, set to "dev" by default.
// ripedbPath - stores the file path for the RIPE DB cache (will be constructed at runtime).
var (
	version    = "dev"
	ripedbPath string
)

// ProgressReader is a wrapper around an io.Reader that displays progress while reading bytes.
type ProgressReader struct {
	Reader    io.Reader // The underlying reader (e.g., response body from HTTP).
	Total     int64     // The total size of the data, used for calculating percentage progress.
	Progress  int64     // The number of bytes read so far.
	Operation string    // A short description of the current operation (e.g., "Downloading").
}

// Read implements the io.Reader interface, updating Progress and printing progress info.
func (pr *ProgressReader) Read(p []byte) (int, error) {
	// Read from the underlying reader into the provided byte slice p.
	n, err := pr.Reader.Read(p)
	// Increment the Progress field by the number of bytes read.
	pr.Progress += int64(n)

	// If the total size is known, display percentage; otherwise just show bytes read.
	if pr.Total > 0 {
		percent := float64(pr.Progress) / float64(pr.Total) * 100
		fmt.Printf("\r%s... %.2f%%", pr.Operation, percent)
	} else {
		fmt.Printf("\r%s... %d bytes", pr.Operation, pr.Progress)
	}
	return n, err
}

func main() {
	// Attempt to get the current user's home directory. If this fails, abort.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return
	}

	// Construct the default path to the cached RIPE DB file:
	// For example, "~/.ripe.db.cache/ripe.db.inetnum"
	ripedbPath = filepath.Join(homeDir, ".ripe.db.cache/ripe.db.inetnum")

	// Check if at least 2 arguments exist (the binary name is the first arg).
	if len(os.Args) < 2 {
		usage()
		return
	}

	// The first actual argument is the command (e.g., "-u", "-dns-acl", etc.).
	cmd := os.Args[1]

	// We use a switch statement to handle each supported command.
	switch cmd {
	case "-h", "--help":
		// Print usage information.
		usage()

	case "-l":
		// Show a list of known country codes and their full names.
		showAvailableCountryCodes()

	case "-v", "--version":
		// Print the current application version string.
		fmt.Printf("version: %s\n", version)

	case "-u":
		// Update (download and extract) the RIPE database to the local cache.
		updateRIPEdb()

	case "-dns-acl":
		// Generate an unfiltered BIND ACL file for the specified country code.
		if len(os.Args) > 2 {
			countryCode := os.Args[2]
			ensureRIPEdb()             // Make sure the RIPE DB file is present.
			createBindACL(countryCode) // Create BIND ACL.
		} else {
			usage()
		}

	case "-dns-acl-f":
		// Generate a filtered BIND ACL file (remove nested subnets).
		if len(os.Args) > 2 {
			countryCode := os.Args[2]
			ensureRIPEdb()
			createBindACLFiltered(countryCode)
		} else {
			usage()
		}

	case "-ovpn":
		// Generate an unfiltered OpenVPN exclude-route file for the given country.
		if len(os.Args) > 2 {
			countryCode := os.Args[2]
			ensureRIPEdb()
			createOpenVPNExclude(countryCode)
		} else {
			usage()
		}

	case "-ovpn-f":
		// Generate a filtered OpenVPN exclude-route file for the given country.
		if len(os.Args) > 2 {
			countryCode := os.Args[2]
			ensureRIPEdb()
			createOpenVPNExcludeFiltered(countryCode)
		} else {
			usage()
		}

	default:
		// If an unknown command is supplied, print usage again.
		usage()
	}
}

// usage prints a help message describing each command-line option.
func usage() {
	fmt.Println(`Usage: chicha-whois <option>
Options:
  -h, --help               Show this help message
  -v, --version            Show the version of this application
  -u                       Update RIPE NCC database. RIPE NCC manages Internet resources for Europe, the Middle East, Central Asia, the Caucasus region, and parts of Russia.
  -l                       Show available country codes

  # Generate DNS Bind ACL (unfiltered / filtered)
  -dns-acl COUNTRYCODE     Generate ACL list for DNS BIND based on country code
  -dns-acl-f COUNTRYCODE   Generate filtered ACL list for DNS BIND based on country code

  # Generate OpenVPN exclude-route list (unfiltered / filtered)
  -ovpn COUNTRYCODE        Generate unfiltered exclude-route list for OpenVPN
  -ovpn-f COUNTRYCODE      Generate filtered exclude-route list for OpenVPN`)
}

// ensureRIPEdb checks if the cached RIPE DB file exists, and if not, calls updateRIPEdb().
func ensureRIPEdb() {
	if _, err := os.Stat(ripedbPath); os.IsNotExist(err) {
		fmt.Println("RIPE database cache not found. Updating...")
		updateRIPEdb()
	}
}

// updateRIPEdb downloads the RIPE database from a publicly available URL and extracts it.
func updateRIPEdb() {
	// This is the location to download the gzipped RIPE database file.
	downloadURL := "https://ftp.ripe.net/ripe/dbase/split/ripe.db.inetnum.gz"

	// Get the current user's home directory to store temporary files.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return
	}

	// Create a temporary file in the home directory for the compressed DB.
	tmpFile, err := os.CreateTemp(homeDir, "ripe.db.inetnum-*.gz")
	if err != nil {
		fmt.Println("Error creating temporary file:", err)
		return
	}
	defer func() {
		// Remove the temporary file once we are done.
		_ = os.Remove(tmpFile.Name())
		fmt.Println("Temporary file removed:", tmpFile.Name())
	}()
	defer tmpFile.Close()

	fmt.Printf("Starting download of RIPE database from %s\n", downloadURL)
	fmt.Printf("Saving to temporary file: %s\n", tmpFile.Name())

	// Perform an HTTP GET request to fetch the RIPE DB.
	resp, err := http.Get(downloadURL)
	if err != nil {
		fmt.Printf("Error downloading RIPE database: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Retrieve the total size from the Content-Length header (may be -1 if unknown).
	totalSize := resp.ContentLength
	if totalSize <= 0 {
		fmt.Println("Unable to determine file size for progress display.")
	} else {
		fmt.Printf("Total file size: %d bytes\n", totalSize)
	}

	// Wrap the response body with a ProgressReader to print download progress.
	progressReader := &ProgressReader{
		Reader:    resp.Body,
		Total:     totalSize,
		Operation: "Downloading",
	}

	// Copy the downloaded bytes from progressReader to tmpFile.
	_, err = io.Copy(tmpFile, progressReader)
	if err != nil {
		fmt.Println("Error writing to temporary file:", err)
		return
	}
	fmt.Println() // Print a newline after final progress percentage.

	fmt.Printf("Extracting %s to %s\n", tmpFile.Name(), ripedbPath)
	// Decompress the .gz file into our final ripedbPath.
	err = gunzipFileWithProgress(tmpFile.Name(), ripedbPath)
	if err != nil {
		fmt.Println("Error decompressing RIPE database:", err)
		return
	}

	fmt.Printf("RIPE database updated successfully at %s\n", ripedbPath)
}

// gunzipFileWithProgress decompresses a .gz file and writes the output to destination.
func gunzipFileWithProgress(source, destination string) error {
	// First, get the size of the .gz file to track extraction progress.
	fi, err := os.Stat(source)
	if err != nil {
		return err
	}
	compressedSize := fi.Size()

	// Ensure the destination directory exists; create if needed.
	dir := filepath.Dir(destination)
	if err = os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory %s: %v", dir, err)
	}

	// Open the compressed source file for reading.
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()

	// Wrap it in a ProgressReader to show extraction progress.
	progressReader := &ProgressReader{
		Reader: file,
		Total:  compressedSize,
	}

	// Create a new gzip reader on top of progressReader.
	gz, err := gzip.NewReader(progressReader)
	if err != nil {
		return err
	}
	defer gz.Close()

	// Create the destination file for the uncompressed data.
	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()

	// Copy from gz to out, thus decompressing the data.
	_, err = io.Copy(out, gz)
	if err != nil {
		return err
	}
	fmt.Println("\nDecompression completed.")
	return nil
}

// createBindACL extracts all CIDRs for a country (unfiltered) and writes them as a BIND ACL.
func createBindACL(countryCode string) {
	fmt.Printf("Creating BIND ACL for country code: %s\n", countryCode)

	// Extract all CIDRs from the RIPE DB for this country.
	ipRanges := extractCountryCIDRs(countryCode, ripedbPath, false)
	if len(ipRanges) == 0 {
		fmt.Printf("No IP ranges found for country code: %s\n", countryCode)
		return
	}

	// Remove duplicates and sort the final list.
	ipRanges = removeDuplicates(ipRanges)
	sort.Strings(ipRanges)

	// Build the text for the BIND ACL. Each CIDR ends with a semicolon in BIND syntax.
	homeDir, _ := os.UserHomeDir()
	aclFilePath := filepath.Join(homeDir, fmt.Sprintf("acl_%s.conf", countryCode))

	var entries []string
	for _, cidr := range ipRanges {
		entries = append(entries, fmt.Sprintf("  %s;", cidr))
	}
	aclContent := fmt.Sprintf("acl \"%s\" {\n%s\n};\n", countryCode, strings.Join(entries, "\n"))

	// Write the ACL string to a file in the user's home directory.
	if err := os.WriteFile(aclFilePath, []byte(aclContent), 0644); err != nil {
		fmt.Printf("Error writing BIND ACL file: %v\n", err)
		return
	}
	fmt.Printf("BIND ACL file created at: %s\n", aclFilePath)
}

// createBindACLFiltered extracts all CIDRs for a country, filters out nested subnets, and writes them as a BIND ACL.
func createBindACLFiltered(countryCode string) {
	fmt.Printf("Creating BIND ACL with filtering for country code: %s\n", countryCode)

	// Extract all CIDRs from the RIPE DB.
	ipRanges := extractCountryCIDRs(countryCode, ripedbPath, false)
	if len(ipRanges) == 0 {
		fmt.Printf("No IP ranges found for country code: %s\n", countryCode)
		return
	}

	// Remove duplicates, then filter out subnets that are contained in larger blocks.
	ipRanges = removeDuplicates(ipRanges)
	ipRanges = filterRedundantCIDRs(ipRanges)
	sort.Strings(ipRanges)

	// Construct the BIND ACL content.
	homeDir, _ := os.UserHomeDir()
	aclFilePath := filepath.Join(homeDir, fmt.Sprintf("acl_%s.conf", countryCode))

	var entries []string
	for _, cidr := range ipRanges {
		entries = append(entries, fmt.Sprintf("  %s;", cidr))
	}
	aclContent := fmt.Sprintf("acl \"%s\" {\n%s\n};\n", countryCode, strings.Join(entries, "\n"))

	// Save the ACL to file.
	if err := os.WriteFile(aclFilePath, []byte(aclContent), 0644); err != nil {
		fmt.Printf("Error writing BIND ACL file: %v\n", err)
		return
	}
	fmt.Printf("Filtered BIND ACL file created at: %s\n", aclFilePath)
}

// createOpenVPNExclude generates an unfiltered list of OpenVPN 'route' lines to exclude a country's IP ranges.
func createOpenVPNExclude(countryCode string) {
	fmt.Printf("Creating OpenVPN exclude-route list for country code: %s\n", countryCode)

	// Get unfiltered CIDRs for this country from the RIPE DB.
	ipRanges := extractCountryCIDRs(countryCode, ripedbPath, false)
	if len(ipRanges) == 0 {
		fmt.Printf("No IP ranges found for country code: %s\n", countryCode)
		return
	}

	// Remove duplicates and sort them for neatness.
	ipRanges = removeDuplicates(ipRanges)
	sort.Strings(ipRanges)

	// We assemble lines for an OpenVPN config.
	// The user wants all traffic to go through VPN, but we add routes to exclude certain subnets.
	var routeLines []string

	routeLines = append(routeLines,
		"# Redirect all traffic through VPN",
		"redirect-gateway def1",
		"",
		fmt.Sprintf("# Exclude %s IPs from VPN", strings.ToUpper(countryCode)),
	)

	// Convert each CIDR to "route <ip> <mask> net_gateway" lines.
	for _, cidr := range ipRanges {
		startIP, netmask, err := cidrToRoute(cidr)
		if err != nil {
			fmt.Printf("Skipping CIDR (%s): %v\n", cidr, err)
			continue
		}
		line := fmt.Sprintf("route %s %s net_gateway", startIP, netmask)
		routeLines = append(routeLines, line)
	}

	// Write them to a .txt file under the user's home directory.
	homeDir, _ := os.UserHomeDir()
	outFilePath := filepath.Join(homeDir, fmt.Sprintf("openvpn_exclude_%s.txt", strings.ToUpper(countryCode)))

	content := strings.Join(routeLines, "\n") + "\n"
	if err := os.WriteFile(outFilePath, []byte(content), 0644); err != nil {
		fmt.Printf("Error writing OpenVPN exclude file: %v\n", err)
		return
	}
	fmt.Printf("OpenVPN exclude-route file created at: %s\n", outFilePath)
}

// createOpenVPNExcludeFiltered generates a filtered list of OpenVPN 'route' lines to exclude a country's IP ranges.
func createOpenVPNExcludeFiltered(countryCode string) {
	fmt.Printf("Creating FILTERED OpenVPN exclude-route list for country code: %s\n", countryCode)

	// Get all country CIDRs from the RIPE DB.
	ipRanges := extractCountryCIDRs(countryCode, ripedbPath, false)
	if len(ipRanges) == 0 {
		fmt.Printf("No IP ranges found for country code: %s\n", countryCode)
		return
	}

	// Remove duplicates, filter out contained subnets, and sort.
	ipRanges = removeDuplicates(ipRanges)
	ipRanges = filterRedundantCIDRs(ipRanges)
	sort.Strings(ipRanges)

	// Build the route lines with some commentary.
	var routeLines []string
	routeLines = append(routeLines,
		"# Redirect all traffic through VPN",
		"redirect-gateway def1",
		"",
		fmt.Sprintf("# Exclude %s IPs from VPN (FILTERED)", strings.ToUpper(countryCode)),
	)

	// Convert each CIDR to "route <ip> <mask> net_gateway".
	for _, cidr := range ipRanges {
		startIP, netmask, err := cidrToRoute(cidr)
		if err != nil {
			fmt.Printf("Skipping CIDR (%s): %v\n", cidr, err)
			continue
		}
		line := fmt.Sprintf("route %s %s net_gateway", startIP, netmask)
		routeLines = append(routeLines, line)
	}

	// Save to the same naming pattern as the unfiltered version, but it's still "openvpn_exclude_<COUNTRY>.txt".
	homeDir, _ := os.UserHomeDir()
	outFilePath := filepath.Join(homeDir, fmt.Sprintf("openvpn_exclude_%s.txt", strings.ToUpper(countryCode)))

	content := strings.Join(routeLines, "\n") + "\n"
	if err := os.WriteFile(outFilePath, []byte(content), 0644); err != nil {
		fmt.Printf("Error writing OpenVPN exclude file: %v\n", err)
		return
	}
	fmt.Printf("Filtered OpenVPN exclude-route file created at: %s\n", outFilePath)
}

// cidrToRoute parses a string like "192.168.1.0/24" into network address ("192.168.1.0") and netmask ("255.255.255.0").
// This is needed for OpenVPN's "route <network> <netmask> net_gateway" format.
func cidrToRoute(cidr string) (string, string, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse CIDR: %v", err)
	}
	// ipNet.IP is the network's starting address. We convert mask to dotted-decimal for OpenVPN.
	networkAddr := ipNet.IP.To4()
	if networkAddr == nil {
		return "", "", fmt.Errorf("IPv6 addresses are not supported in this example")
	}
	netmask := ipMaskToDotted(ipNet.Mask)
	return networkAddr.String(), netmask, nil
}

// ipMaskToDotted converts a net.IPMask (e.g. /24) to its dotted-decimal string (e.g. "255.255.255.0").
func ipMaskToDotted(mask net.IPMask) string {
	if len(mask) != 4 {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}

// extractCountryCIDRs scans the local RIPE DB file for inetnum blocks belonging to the specified country code.
// It returns a list of CIDR strings (e.g. "192.168.0.0/16").
func extractCountryCIDRs(countryCode, dbPath string, debugPrint bool) []string {
	// Open the RIPE DB file for reading.
	file, err := os.Open(dbPath)
	if err != nil {
		fmt.Println("Error opening RIPE database:", err)
		return nil
	}
	defer file.Close()

	// We always uppercase the user's input so it matches entries in DB.
	countryCode = strings.ToUpper(countryCode)

	scanner := bufio.NewScanner(file)
	var ipRanges []string
	var blockLines []string

	// We read block-by-block: each block ends on an empty line in the RIPE DB.
	for {
		blockLines = nil
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				// A blank line signals the end of a block.
				break
			}
			blockLines = append(blockLines, line)
		}

		// If no lines were read, we've reached the end of the file.
		if len(blockLines) == 0 {
			break
		}

		// We'll look in each block for lines beginning with "inetnum:" and "country:".
		var inetnumLine, countryLine string
		for _, line := range blockLines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "inetnum:") {
				inetnumLine = line
			} else if strings.HasPrefix(line, "country:") {
				countryLine = line
			}
		}

		// If both inetnum and country are present in this block, we check if it matches our country code.
		if inetnumLine != "" && countryLine != "" {
			fields := strings.Fields(countryLine)
			if len(fields) >= 2 {
				blockCountryCode := strings.ToUpper(fields[1])
				if blockCountryCode == countryCode {
					// This block belongs to the requested country.
					inetnumFields := strings.Fields(inetnumLine)
					if len(inetnumFields) >= 2 {
						// inetnum: <startIP> - <endIP>
						ipRangeStr := strings.Join(inetnumFields[1:], " ")
						ipRangeParts := strings.Split(ipRangeStr, "-")
						if len(ipRangeParts) == 2 {
							start := strings.TrimSpace(ipRangeParts[0])
							end := strings.TrimSpace(ipRangeParts[1])

							if debugPrint {
								fmt.Printf("Found inetnum entry: %s - %s\n", start, end)
							}

							// Convert the range to a single CIDR.
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

// generateCIDR attempts to convert a start IP and end IP into a single CIDR notation (e.g., 192.168.0.0/24).
// NOTE: This only works correctly if the range aligns with a power-of-two subnet boundary.
func generateCIDR(startIPStr, endIPStr string) string {
	// Parse IP addresses into net.IP and force IPv4 for simplicity.
	startIP := net.ParseIP(startIPStr).To4()
	endIP := net.ParseIP(endIPStr).To4()
	if startIP == nil || endIP == nil {
		fmt.Printf("Error: Invalid IP address range: %s - %s\n", startIPStr, endIPStr)
		return ""
	}

	// Convert them to uint32 to manipulate bits.
	start := binary.BigEndian.Uint32(startIP)
	end := binary.BigEndian.Uint32(endIP)

	// XOR to find differing bits, then use bits.Len32() to determine how many bits are different.
	diff := start ^ end
	prefixLength := 32 - bits.Len32(diff)

	// Zero out the bits after prefixLength to get the network address.
	network := start &^ ((1 << (32 - prefixLength)) - 1)

	// Convert the integer back to a dotted-quad IP.
	networkIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(networkIP, network)

	cidr := fmt.Sprintf("%s/%d", networkIP.String(), prefixLength)
	return cidr
}

// filterRedundantCIDRs iterates through a list of CIDRs, removing those fully contained by any larger CIDR.
func filterRedundantCIDRs(cidrs []string) []string {
	var parsedCIDRs []*net.IPNet

	// Parse each CIDR string into a *net.IPNet.
	for _, cidrStr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidrStr)
		if err != nil {
			fmt.Printf("Error parsing CIDR %s: %v\n", cidrStr, err)
			continue
		}
		parsedCIDRs = append(parsedCIDRs, ipNet)
	}

	// Sort by (prefix length ascending) and then by IP address ascending.
	// This ensures that bigger subnets (e.g. /16) come before smaller ones (/24).
	sort.Slice(parsedCIDRs, func(i, j int) bool {
		onesI, bitsI := parsedCIDRs[i].Mask.Size()
		onesJ, bitsJ := parsedCIDRs[j].Mask.Size()

		// Compare the total bits (IPv4 vs IPv6).
		if bitsI != bitsJ {
			return bitsI < bitsJ
		}
		// Then compare prefix lengths.
		if onesI != onesJ {
			return onesI < onesJ
		}
		// Finally compare the IP addresses themselves.
		return bytes.Compare(parsedCIDRs[i].IP, parsedCIDRs[j].IP) < 0
	})

	var keptCIDRs []*net.IPNet

	// For each candidate, check if it is contained in any CIDR already kept.
	for _, candidate := range parsedCIDRs {
		redundant := false
		for _, keeper := range keptCIDRs {
			if cidrContains(keeper, candidate) {
				redundant = true
				fmt.Printf("Filtered out redundant CIDR: %s (contained in %s)\n", candidate.String(), keeper.String())
				break
			}
		}
		if !redundant {
			keptCIDRs = append(keptCIDRs, candidate)
		}
	}

	// Convert the kept CIDRs back into string form.
	var results []string
	for _, net := range keptCIDRs {
		results = append(results, net.String())
	}
	return results
}

// cidrContains checks if 'inner' is fully contained within 'outer' (they are both IPv4 subnets).
func cidrContains(outer, inner *net.IPNet) bool {
	// First ensure the first IP of inner is in outer.
	if !outer.Contains(inner.IP) {
		return false
	}
	// Compute the last IP of inner's range, and ensure that is also in outer.
	innerLast := lastIP(inner)
	return outer.Contains(innerLast)
}

// lastIP returns the broadcast (last) address in a given subnet range.
func lastIP(ipNet *net.IPNet) net.IP {
	ip := ipNet.IP.To4()
	if ip == nil {
		// We skip IPv6 in this example, but you could add support if needed.
		return nil
	}
	mask := ipNet.Mask
	network := ip.Mask(mask)
	broadcast := make(net.IP, len(network))

	// For each byte of the mask, invert the bits to find the broadcast address.
	for i := 0; i < len(network); i++ {
		broadcast[i] = network[i] | ^mask[i]
	}
	return broadcast
}

// removeDuplicates discards duplicate strings in a slice while preserving order.
func removeDuplicates(elements []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, e := range elements {
		if !seen[e] {
			seen[e] = true
			result = append(result, e)
		}
	}
	return result
}

// showAvailableCountryCodes prints known country codes and their names in alphabetical order by country name. RIPE NCC (Réseaux IP Européens Network Coordination Centre) is one of the five Regional Internet Registries (RIRs) and primarily serves Europe, the Middle East, and parts of Central Asia. Here’s the filtered list of countries that are within the RIPE NCC service region, which overlaps with the European region (including some non-EU countries) and neighboring areas:

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

	// We'll create a slice of structs to sort by the country Name value.
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

	// Sort alphabetically by the Name field.
	sort.Slice(countryList, func(i, j int) bool {
		return countryList[i].Name < countryList[j].Name
	})

	// Print out each code and name pair in sorted order.
	fmt.Println("Available country codes and names (sorted by name):")
	for _, c := range countryList {
		fmt.Printf("%s - %s\n", c.Code, c.Name)
	}
}
