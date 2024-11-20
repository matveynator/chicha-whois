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

var (
	// Global variable storing the path to the RIPE database cache file
	version    = "dev"
	ripedbPath string
)

// ProgressReader is a wrapper around an io.Reader that tracks the number of bytes read and displays progress.
type ProgressReader struct {
	Reader    io.Reader // The underlying reader we're wrapping (e.g., response body).
	Total     int64     // Total size of the data to be read (for progress calculation).
	Progress  int64     // Amount of data read so far.
	Operation string    // Description of the operation (e.g., "Downloading").
}

// Read reads data from the underlying reader and updates the progress.
func (pr *ProgressReader) Read(p []byte) (int, error) {
	// Read data into p from the underlying reader.
	n, err := pr.Reader.Read(p)
	// Update the total number of bytes read so far.
	pr.Progress += int64(n)
	// If total size is known, calculate and display the progress percentage.
	if pr.Total > 0 {
		percent := float64(pr.Progress) / float64(pr.Total) * 100
		fmt.Printf("\r%s... %.2f%%", pr.Operation, percent)
	} else {
		// If total size is unknown, display bytes read.
		fmt.Printf("\r%s... %d bytes", pr.Operation, pr.Progress)
	}
	return n, err
}

func main() {
	// Retrieve the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return
	}
	// Construct the path to the RIPE database cache
	ripedbPath = filepath.Join(homeDir, ".ripe.db.cache/ripe.db.inetnum")

	// Ensure there is a command-line argument to process
	if len(os.Args) < 2 {
		usage()
		return
	}

	// Handle the specified command
	cmd := os.Args[1]
	switch cmd {
	case "-h", "--help":
		usage()
	case "-l":
		// Show available country codes
		showAvailableCountryCodes()
	case "-v", "--version":
		fmt.Printf("version: %s\n", version)
	case "-u":
		// Update the RIPE database
		updateRIPEdb()
	case "-dns-acl":
		// Generate ACL list for BIND based on the provided country code
		if len(os.Args) > 2 {
			countryCode := os.Args[2]
			// Ensure the RIPE database cache exists; update if missing
			if _, err := os.Stat(ripedbPath); os.IsNotExist(err) {
				fmt.Println("RIPE database cache not found. Updating...")
				updateRIPEdb()
			}
			createBindACL(countryCode)
		} else {
			usage()
		}
	case "-dns-acl-f":
		// Generate ACL list for BIND with filtering of redundant subnets
		if len(os.Args) > 2 {
			countryCode := os.Args[2]
			// Ensure the RIPE database cache exists; update if missing
			if _, err := os.Stat(ripedbPath); os.IsNotExist(err) {
				fmt.Println("RIPE database cache not found. Updating...")
				updateRIPEdb()
			}
			createBindACLFiltered(countryCode)
		} else {
			usage()
		}
	default:
		// Show usage for unknown commands
		usage()
	}
}

func usage() {
	fmt.Println(`Usage: chicha-whois <option>
Options:
  -h, --help              Show this help message
  -v, --version           Show the version of this application
  -u                      Update RIPE database
  -dns-acl COUNTRYCODE    Generate ACL list for DNS BIND based on country code
  -dns-acl-f COUNTRYCODE  Generate filtered ACL list for DNS BIND based on country code
  -l                      Show available country codes`)
}
func updateRIPEdb() {
	// Updates the RIPE database cache with detailed progress display during download and extraction.

	// Define the URL to download the RIPE database from.
	downloadURL := "https://ftp.ripe.net/ripe/dbase/split/ripe.db.inetnum.gz"

	// Retrieve the user's home directory to store temporary and output files.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return
	}

	// Create a temporary file in the home directory for storing the downloaded RIPE database.
	tmpFile, err := os.CreateTemp(homeDir, "ripe.db.inetnum-*.gz")
	if err != nil {
		fmt.Println("Error creating temporary file:", err)
		return
	}
	// Ensure the temporary file is closed and deleted after we're done.
	defer func() {
		err := os.Remove(tmpFile.Name())
		if err != nil {
			fmt.Println("Warning: failed to remove temporary file:", tmpFile.Name())
		} else {
			fmt.Println("Temporary file removed:", tmpFile.Name())
		}
	}()
	defer tmpFile.Close()

	// Inform the user about the download initiation.
	fmt.Printf("Starting download of RIPE database from %s\n", downloadURL)
	fmt.Printf("Saving to temporary file: %s\n", tmpFile.Name())

	// Initiate the HTTP GET request to download the RIPE database.
	resp, err := http.Get(downloadURL)
	if err != nil {
		fmt.Printf("Error downloading RIPE database from %s: %v\n", downloadURL, err)
		return
	}
	// Ensure the response body is closed after reading.
	defer resp.Body.Close()

	// Get the total size of the file from the Content-Length header for progress calculation.
	totalSize := resp.ContentLength
	if totalSize <= 0 {
		fmt.Println("Unable to determine file size for progress display")
	} else {
		fmt.Printf("Total file size: %d bytes\n", totalSize)
	}

	// Create a progress reader to wrap the response body and track the download progress.
	progressReader := &ProgressReader{
		Reader:    resp.Body,     // The underlying reader (response body).
		Total:     totalSize,     // Total size of the file for progress calculation.
		Operation: "Downloading", // Operation description for progress display.
	}

	// Copy the response body to the temporary file, reading through the progress reader.
	_, err = io.Copy(tmpFile, progressReader)
	if err != nil {
		fmt.Println("Error writing to temporary file:", err)
		return
	}
	fmt.Println() // Move to the next line after progress display.

	// Inform the user about the extraction initiation.
	fmt.Printf("Extracting %s to %s\n", tmpFile.Name(), ripedbPath)

	// Decompress the Gzip file and save it to the final path, with progress display.
	err = gunzipFileWithProgress(tmpFile.Name(), ripedbPath)
	if err != nil {
		fmt.Println("Error decompressing RIPE database:", err)
		return
	}

	// Inform the user that the RIPE database has been updated successfully.
	fmt.Printf("RIPE database updated successfully at %s\n", ripedbPath)
}

func gunzipFileWithProgress(source, destination string) error {
	// Decompresses a Gzip file and saves it to the destination with progress display during extraction

	// Get the size of the compressed file (source)
	fi, err := os.Stat(source)
	if err != nil {
		return err
	}
	compressedSize := fi.Size()

	// Create the destination directory if it doesn't exist
	dir := filepath.Dir(destination) // Extract directory path
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating directory %s: %v", dir, err)
	}

	// Open the compressed source file
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a progress reader to wrap the source file and track the decompression progress
	progressReader := &ProgressReader{
		Reader: file,           // The underlying reader (compressed file)
		Total:  compressedSize, // Total size of the compressed file
	}

	// Create a Gzip reader using the progress reader
	gz, err := gzip.NewReader(progressReader)
	if err != nil {
		return err
	}
	defer gz.Close()

	// Open the destination file for writing the decompressed data
	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()

	// Copy the decompressed data from the Gzip reader to the destination file
	_, err = io.Copy(out, gz)
	if err != nil {
		return err
	}
	fmt.Println("\nDecompression completed.") // Indicate that decompression is complete
	return nil
}

func createBindACL(countryCode string) {
	// Generates a BIND ACL for the specified country code using data from the RIPE database.
	fmt.Printf("Creating BIND ACL for country code: %s\n", countryCode)

	// Open the RIPE database file.
	file, err := os.Open(ripedbPath)
	if err != nil {
		fmt.Println("Error opening RIPE database:", err)
		return
	}
	defer file.Close()

	// Prepare to read the file block by block.
	scanner := bufio.NewScanner(file)
	var ipRanges []string
	countryCode = strings.ToUpper(countryCode) // Ensure country code is in uppercase for comparison.

	// Variables to hold block data.
	var blockLines []string

	for {
		blockLines = nil // Reset the block lines for each new block.

		// Read a block of lines until an empty line is encountered.
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				// Empty line indicates the end of a block.
				break
			}
			blockLines = append(blockLines, line)
		}

		// If no lines were read, we've reached the end of the file.
		if len(blockLines) == 0 {
			break
		}

		// Initialize variables to store inetnum and country data from the block.
		var inetnumLine, countryLine string

		// Process each line in the block to find inetnum and country information.
		for _, line := range blockLines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "inetnum:") {
				inetnumLine = line
			} else if strings.HasPrefix(line, "country:") {
				countryLine = line
			}
		}

		// If both inetnum and country data are found, proceed to process the block.
		if inetnumLine != "" && countryLine != "" {
			// Extract the country code from the country line.
			fields := strings.Fields(countryLine)
			if len(fields) >= 2 {
				blockCountryCode := strings.ToUpper(fields[1])
				if blockCountryCode == countryCode {
					// Extract the IP range from the inetnum line.
					inetnumFields := strings.Fields(inetnumLine)
					if len(inetnumFields) >= 2 {
						ipRangeStr := strings.Join(inetnumFields[1:], " ")
						// The IP range string is expected to be in the format "startIP - endIP".
						ipRangeParts := strings.Split(ipRangeStr, "-")
						if len(ipRangeParts) == 2 {
							ipRangeStart := strings.TrimSpace(ipRangeParts[0])
							ipRangeEnd := strings.TrimSpace(ipRangeParts[1])

							// Log the found inetnum entry.
							fmt.Printf("Found inetnum entry: %s - %s\n", ipRangeStart, ipRangeEnd)

							// Generate a single CIDR block from the IP range.
							cidr := generateCIDR(ipRangeStart, ipRangeEnd)

							if cidr != "" {
								fmt.Printf("Converted to CIDR: %s\n", cidr)
								// Add the generated CIDR to the collection of IP ranges.
								ipRanges = append(ipRanges, cidr)
							}
						}
					}
				}
			}
		}

		// Continue to the next block.
	}

	// Check if any IP ranges were found.
	if len(ipRanges) == 0 {
		fmt.Printf("No IP ranges found for country code: %s\n", countryCode)
		return
	}

	// Remove duplicate CIDR entries.
	ipRanges = removeDuplicates(ipRanges)

	// Sort the IP ranges.
	sort.Strings(ipRanges)

	// Get the user's home directory to save the file locally.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return
	}
	// Create the ACL file path in the user's local directory.
	aclFilePath := filepath.Join(homeDir, fmt.Sprintf("acl_%s.conf", countryCode))

	// Create the ACL file for BIND.
	var entries []string
	for _, cidr := range ipRanges {
		entries = append(entries, fmt.Sprintf("  %s;", cidr)) // Format each CIDR as a BIND ACL entry.
	}
	aclContent := fmt.Sprintf("acl \"%s\" {\n%s\n};\n", countryCode, strings.Join(entries, "\n"))

	// Write the ACL content to the specified file.
	err = os.WriteFile(aclFilePath, []byte(aclContent), 0644)
	if err != nil {
		fmt.Printf("Error writing BIND ACL file: %v\n", err)
		return
	}
	fmt.Printf("BIND ACL file created at: %s\n", aclFilePath)
}

func createBindACLFiltered(countryCode string) {
	// Generates a BIND ACL for the specified country code, filtering out redundant subnets.
	fmt.Printf("Creating BIND ACL with filtering for country code: %s\n", countryCode)

	// Open the RIPE database file.
	file, err := os.Open(ripedbPath)
	if err != nil {
		fmt.Println("Error opening RIPE database:", err)
		return
	}
	defer file.Close()

	// Prepare to read the file block by block.
	scanner := bufio.NewScanner(file)
	var ipRanges []string
	countryCode = strings.ToUpper(countryCode) // Ensure country code is in uppercase for comparison.

	// Variables to hold block data.
	var blockLines []string

	for {
		blockLines = nil // Reset the block lines for each new block.

		// Read a block of lines until an empty line is encountered.
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				// Empty line indicates the end of a block.
				break
			}
			blockLines = append(blockLines, line)
		}

		// If no lines were read, we've reached the end of the file.
		if len(blockLines) == 0 {
			break
		}

		// Initialize variables to store inetnum and country data from the block.
		var inetnumLine, countryLine string

		// Process each line in the block to find inetnum and country information.
		for _, line := range blockLines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "inetnum:") {
				inetnumLine = line
			} else if strings.HasPrefix(line, "country:") {
				countryLine = line
			}
		}

		// If both inetnum and country data are found, proceed to process the block.
		if inetnumLine != "" && countryLine != "" {
			// Extract the country code from the country line.
			fields := strings.Fields(countryLine)
			if len(fields) >= 2 {
				blockCountryCode := strings.ToUpper(fields[1])
				if blockCountryCode == countryCode {
					// Extract the IP range from the inetnum line.
					inetnumFields := strings.Fields(inetnumLine)
					if len(inetnumFields) >= 2 {
						ipRangeStr := strings.Join(inetnumFields[1:], " ")
						// The IP range string is expected to be in the format "startIP - endIP".
						ipRangeParts := strings.Split(ipRangeStr, "-")
						if len(ipRangeParts) == 2 {
							ipRangeStart := strings.TrimSpace(ipRangeParts[0])
							ipRangeEnd := strings.TrimSpace(ipRangeParts[1])

							// Log the found inetnum entry.
							fmt.Printf("Found inetnum entry: %s - %s\n", ipRangeStart, ipRangeEnd)

							// Generate a single CIDR block from the IP range.
							cidr := generateCIDR(ipRangeStart, ipRangeEnd)

							if cidr != "" {
								fmt.Printf("Converted to CIDR: %s\n", cidr)
								// Add the generated CIDR to the collection of IP ranges.
								ipRanges = append(ipRanges, cidr)
							}
						}
					}
				}
			}
		}

		// Continue to the next block.
	}

	// Check if any IP ranges were found.
	if len(ipRanges) == 0 {
		fmt.Printf("No IP ranges found for country code: %s\n", countryCode)
		return
	}

	// Remove duplicate CIDR entries.
	ipRanges = removeDuplicates(ipRanges)

	// Filter out redundant CIDRs.
	ipRanges = filterRedundantCIDRs(ipRanges)

	// Sort the IP ranges.
	sort.Strings(ipRanges)

	// Get the user's home directory to save the file locally.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return
	}
	// Create the ACL file path in the user's local directory.
	aclFilePath := filepath.Join(homeDir, fmt.Sprintf("acl_%s.conf", countryCode))

	// Create the ACL file for BIND.
	var entries []string
	for _, cidr := range ipRanges {
		entries = append(entries, fmt.Sprintf("  %s;", cidr)) // Format each CIDR as a BIND ACL entry.
	}
	aclContent := fmt.Sprintf("acl \"%s\" {\n%s\n};\n", countryCode, strings.Join(entries, "\n"))

	// Write the ACL content to the specified file.
	err = os.WriteFile(aclFilePath, []byte(aclContent), 0644)
	if err != nil {
		fmt.Printf("Error writing BIND ACL file: %v\n", err)
		return
	}
	fmt.Printf("Filtered BIND ACL file created at: %s\n", aclFilePath)
}

func generateCIDR(startIPStr, endIPStr string) string {
	// Parses IP addresses
	startIP := net.ParseIP(startIPStr).To4()
	endIP := net.ParseIP(endIPStr).To4()
	if startIP == nil || endIP == nil {
		fmt.Printf("Error: Invalid IP address range: %s - %s\n", startIPStr, endIPStr)
		return ""
	}

	// Converts IPs to uint32 for calculations
	start := binary.BigEndian.Uint32(startIP)
	end := binary.BigEndian.Uint32(endIP)

	// Calculate the number of bits to cover the range
	diff := start ^ end
	prefixLength := 32 - bits.Len32(diff)

	// Calculate the network address
	network := start &^ ((1 << (32 - prefixLength)) - 1)

	// Convert network address back to IP
	networkIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(networkIP, network)

	// Format CIDR notation
	cidr := fmt.Sprintf("%s/%d", networkIP.String(), prefixLength)
	return cidr
}

func filterRedundantCIDRs(cidrs []string) []string {
	// Filters out CIDRs that are fully contained within other CIDRs in the list.
	var result []string
	// Parse all CIDRs into net.IPNet structures
	parsedCIDRs := make([]*net.IPNet, 0, len(cidrs))
	for _, cidrStr := range cidrs {
		_, cidrNet, err := net.ParseCIDR(cidrStr)
		if err != nil {
			fmt.Printf("Error parsing CIDR %s: %v\n", cidrStr, err)
			continue
		}
		parsedCIDRs = append(parsedCIDRs, cidrNet)
	}

	// Sort the CIDRs by prefix length ascending (shorter prefixes first), then by IP
	sort.Slice(parsedCIDRs, func(i, j int) bool {
		onesI, bitsI := parsedCIDRs[i].Mask.Size()
		onesJ, bitsJ := parsedCIDRs[j].Mask.Size()
		// Ensure the same address family
		if bitsI != bitsJ {
			return bitsI < bitsJ
		}
		if onesI != onesJ {
			return onesI < onesJ
		}
		// If prefix lengths are equal, sort by IP address
		return bytes.Compare(parsedCIDRs[i].IP, parsedCIDRs[j].IP) < 0
	})

	// Prepare a list to store the CIDRs we keep
	keptCIDRs := []*net.IPNet{}

	// Iterate over the sorted CIDRs
	for _, cidrNet := range parsedCIDRs {
		redundant := false
		// Check if current CIDR is contained in any of the kept CIDRs
		for _, keptNet := range keptCIDRs {
			if cidrContains(keptNet, cidrNet) {
				redundant = true
				fmt.Printf("Filtered out redundant CIDR: %s (contained in %s)\n", cidrNet.String(), keptNet.String())
				break
			}
		}
		if !redundant {
			keptCIDRs = append(keptCIDRs, cidrNet)
		}
	}

	// Convert the kept CIDRs back to strings
	for _, cidrNet := range keptCIDRs {
		result = append(result, cidrNet.String())
	}

	return result
}

func cidrContains(outer, inner *net.IPNet) bool {
	// Checks if 'inner' CIDR is fully contained within 'outer' CIDR
	// First, check if the outer network contains the first IP of the inner network
	if !outer.Contains(inner.IP) {
		return false
	}

	// Calculate the last IP of the inner network
	innerLastIP := lastIP(inner)
	// Check if the outer network contains the last IP of the inner network
	return outer.Contains(innerLastIP)
}

func lastIP(ipNet *net.IPNet) net.IP {
	// Calculates the last IP address in the CIDR block
	ip := ipNet.IP.To4()
	if ip == nil {
		// IPv6 not supported in this code
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

func removeDuplicates(elements []string) []string {
	// Removes duplicate strings from a slice.
	encountered := map[string]bool{}
	result := []string{}

	for _, v := range elements {
		if !encountered[v] {
			encountered[v] = true
			result = append(result, v)
		}
	}
	return result
}

func showAvailableCountryCodes() {

	countries := map[string]string{
		"AD": "Andorra", "AE": "United Arab Emirates", "AF": "Afghanistan", "AG": "Antigua and Barbuda",
		"AI": "Anguilla", "AL": "Albania", "AM": "Armenia", "AO": "Angola", "AQ": "Antarctica", "AR": "Argentina",
		"AS": "American Samoa", "AT": "Austria", "AU": "Australia", "AW": "Aruba", "AX": "Aland Islands",
		"AZ": "Azerbaijan", "BA": "Bosnia and Herzegovina", "BB": "Barbados", "BD": "Bangladesh", "BE": "Belgium",
		"BF": "Burkina Faso", "BG": "Bulgaria", "BH": "Bahrain", "BI": "Burundi", "BJ": "Benin", "BL": "Saint Barthélemy",
		"BM": "Bermuda", "BN": "Brunei", "BO": "Bolivia", "BQ": "Bonaire, Sint Eustatius, and Saba", "BR": "Brazil",
		"BS": "Bahamas", "BT": "Bhutan", "BV": "Bouvet Island", "BW": "Botswana", "BY": "Belarus", "BZ": "Belize",
		"CA": "Canada", "CC": "Cocos (Keeling) Islands", "CD": "Congo (Kinshasa)", "CF": "Central African Republic",
		"CG": "Congo (Brazzaville)", "CH": "Switzerland", "CI": "Côte d'Ivoire", "CK": "Cook Islands", "CL": "Chile",
		"CM": "Cameroon", "CN": "China", "CO": "Colombia", "CR": "Costa Rica", "CU": "Cuba", "CV": "Cape Verde",
		"CW": "Curaçao", "CX": "Christmas Island", "CY": "Cyprus", "CZ": "Czech Republic", "DE": "Germany",
		"DJ": "Djibouti", "DK": "Denmark", "DM": "Dominica", "DO": "Dominican Republic", "DZ": "Algeria",
		"EC": "Ecuador", "EE": "Estonia", "EG": "Egypt", "EH": "Western Sahara", "ER": "Eritrea", "ES": "Spain",
		"ET": "Ethiopia", "FI": "Finland", "FJ": "Fiji", "FK": "Falkland Islands", "FM": "Micronesia",
		"FO": "Faroe Islands", "FR": "France", "GA": "Gabon", "GB": "United Kingdom", "GD": "Grenada",
		"GE": "Georgia", "GF": "French Guiana", "GG": "Guernsey", "GH": "Ghana", "GI": "Gibraltar",
		"GL": "Greenland", "GM": "Gambia", "GN": "Guinea", "GP": "Guadeloupe", "GQ": "Equatorial Guinea",
		"GR": "Greece", "GT": "Guatemala", "GU": "Guam", "GW": "Guinea-Bissau", "GY": "Guyana",
		"HK": "Hong Kong", "HM": "Heard Island and McDonald Islands", "HN": "Honduras", "HR": "Croatia",
		"HT": "Haiti", "HU": "Hungary", "ID": "Indonesia", "IE": "Ireland", "IL": "Israel", "IM": "Isle of Man",
		"IN": "India", "IO": "British Indian Ocean Territory", "IQ": "Iraq", "IR": "Iran", "IS": "Iceland",
		"IT": "Italy", "JE": "Jersey", "JM": "Jamaica", "JO": "Jordan", "JP": "Japan", "KE": "Kenya",
		"KG": "Kyrgyzstan", "KH": "Cambodia", "KI": "Kiribati", "KM": "Comoros", "KN": "Saint Kitts and Nevis",
		"KP": "North Korea", "KR": "South Korea", "KW": "Kuwait", "KY": "Cayman Islands", "KZ": "Kazakhstan",
		"LA": "Laos", "LB": "Lebanon", "LC": "Saint Lucia", "LI": "Liechtenstein", "LK": "Sri Lanka",
		"LR": "Liberia", "LS": "Lesotho", "LT": "Lithuania", "LU": "Luxembourg", "LV": "Latvia", "LY": "Libya",
		"MA": "Morocco", "MC": "Monaco", "MD": "Moldova", "ME": "Montenegro", "MF": "Saint Martin", "MG": "Madagascar",
		"MH": "Marshall Islands", "MK": "North Macedonia", "ML": "Mali", "MM": "Myanmar (Burma)", "MN": "Mongolia",
		"MO": "Macao", "MP": "Northern Mariana Islands", "MQ": "Martinique", "MR": "Mauritania", "MS": "Montserrat",
		"MT": "Malta", "MU": "Mauritius", "MV": "Maldives", "MW": "Malawi", "MX": "Mexico", "MY": "Malaysia",
		"MZ": "Mozambique", "NA": "Namibia", "NC": "New Caledonia", "NE": "Niger", "NF": "Norfolk Island",
		"NG": "Nigeria", "NI": "Nicaragua", "NL": "Netherlands", "NO": "Norway", "NP": "Nepal", "NR": "Nauru",
		"NU": "Niue", "NZ": "New Zealand", "OM": "Oman", "PA": "Panama", "PE": "Peru", "PF": "French Polynesia",
		"PG": "Papua New Guinea", "PH": "Philippines", "PK": "Pakistan", "PL": "Poland", "PM": "Saint Pierre and Miquelon",
		"PN": "Pitcairn Islands", "PR": "Puerto Rico", "PT": "Portugal", "PW": "Palau", "PY": "Paraguay",
		"QA": "Qatar", "RE": "Réunion", "RO": "Romania", "RS": "Serbia", "RU": "Russia", "RW": "Rwanda",
		"SA": "Saudi Arabia", "SB": "Solomon Islands", "SC": "Seychelles", "SD": "Sudan", "SE": "Sweden",
		"SG": "Singapore", "SH": "Saint Helena", "SI": "Slovenia", "SJ": "Svalbard and Jan Mayen", "SK": "Slovakia",
		"SL": "Sierra Leone", "SM": "San Marino", "SN": "Senegal", "SO": "Somalia", "SR": "Suriname",
		"SS": "South Sudan", "ST": "São Tomé and Príncipe", "SV": "El Salvador", "SX": "Sint Maarten", "SY": "Syria",
		"SZ": "Eswatini", "TC": "Turks and Caicos Islands", "TD": "Chad", "TF": "French Southern Territories",
		"TG": "Togo", "TH": "Thailand", "TJ": "Tajikistan", "TK": "Tokelau", "TL": "Timor-Leste", "TM": "Turkmenistan",
		"TN": "Tunisia", "TO": "Tonga", "TR": "Turkey", "TT": "Trinidad and Tobago", "TV": "Tuvalu",
		"TZ": "Tanzania", "UA": "Ukraine", "UG": "Uganda", "UM": "United States Minor Outlying Islands",
		"US": "United States", "UY": "Uruguay", "UZ": "Uzbekistan", "VA": "Vatican City", "VC": "Saint Vincent and the Grenadines",
		"VE": "Venezuela", "VG": "British Virgin Islands", "VI": "United States Virgin Islands", "VN": "Vietnam",
		"VU": "Vanuatu", "WF": "Wallis and Futuna", "WS": "Samoa", "YE": "Yemen", "YT": "Mayotte", "ZA": "South Africa",
		"ZM": "Zambia", "ZW": "Zimbabwe",
	}

	// Extract and sort country names
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

	// Sort the list by country names
	sort.Slice(countryList, func(i, j int) bool {
		return countryList[i].Name < countryList[j].Name
	})

	// Display sorted country codes and names
	fmt.Println("Available country codes and their names (sorted by country name):")
	for _, country := range countryList {
		fmt.Printf("%s - %s\n", country.Code, country.Name)
	}
}
