package main

import (
	"bufio"         // Used for reading text from a file efficiently
	"encoding/binary"
	"compress/gzip" // Provides Gzip decompression capabilities
	"fmt"           // Standard formatting and printing functions
	"io"            // Provides input/output primitives
	"net"           // Network-related functions, e.g., IP address parsing
	"net/http"      // HTTP client capabilities
	"os"            // Operating system functions for file handling
	"os/exec"       // Executing external commands
	"path/filepath" // Provides path manipulation functions
	"regexp"        // Regular expression matching
	"sort"          // Sorting functions
	"strings"       // String manipulation functions
)

var (
	// Global variable storing the path to the RIPE database cache file
	ripedbPath string
	// File path for the Nginx TestCookie whitelist
	tcwhitelistPath = "/etc/nginx/testcookie_whitelist.conf"
	// File path for the general Nginx whitelist
	nginxwhitelistPath = "/etc/nginx/whitelist.conf"
)

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
	case "-u":
		// Update the RIPE database
		updateRIPEdb()
	case "-n":
		// Produce a whitelist for Nginx with the specified name
		if len(os.Args) > 2 {
			searchName := os.Args[2]
			// Ensure the RIPE database cache exists; update if missing
			if _, err := os.Stat(ripedbPath); os.IsNotExist(err) {
				fmt.Println("RIPE database cache not found. Updating...")
				updateRIPEdb()
			}
			whitelistNginx(searchName)
		} else {
			usage()
		}
	case "-t":
		// Produce a TestCookie whitelist
		if len(os.Args) > 2 {
			searchName := os.Args[2]
			// Ensure the RIPE database cache exists; update if missing
			if _, err := os.Stat(ripedbPath); os.IsNotExist(err) {
				fmt.Println("RIPE database cache not found. Updating...")
				updateRIPEdb()
			}
			whitelistTestCookie(searchName)
		} else {
			usage()
		}
	case "-tor":
		// Blacklist TOR addresses
		if _, err := os.Stat(ripedbPath); os.IsNotExist(err) {
			fmt.Println("RIPE database cache not found. Updating...")
			updateRIPEdb()
		}
		blacklistTOR()
	case "-acl":
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
	default:
		// Show usage for unknown commands
		usage()
	}
}

func usage() {
	// Prints usage instructions for the tool
	fmt.Println(`Usage: ripe-country-list <option>
	Options:
	-h, --help    Show this help message
	-l            Show available country codes
	-u            Update RIPE database
	-n NAME       Produce whitelist for Nginx
	-t NAME       Produce whitelist for TestCookie
	-tor          Produce blacklist of TOR addresses for Nginx
	-acl COUNTRY  Generate ACL list for BIND based on country code`)
}

func updateRIPEdb() {
	// Updates the RIPE database cache
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return
	}

	// Create a temporary file for storing the downloaded RIPE database
	tmpFile, err := os.CreateTemp(homeDir, "ripe.db.inetnum-*.gz")
	if err != nil {
		fmt.Println("Error creating temporary file:", err)
		return
	}
	defer func() {
		// Delete the temporary file after processing
		err := os.Remove(tmpFile.Name())
		if err != nil {
			fmt.Println("Warning: failed to remove temporary file:", tmpFile.Name())
		} else {
			fmt.Println("Temporary file removed:", tmpFile.Name())
		}
	}()
	defer tmpFile.Close()

	// Download the RIPE database from the specified URL
	resp, err := http.Get("https://ftp.ripe.net/ripe/dbase/split/ripe.db.inetnum.gz")
	if err != nil {
		fmt.Println("Error downloading RIPE database:", err)
		return
	}
	defer resp.Body.Close()

	_, err = io.Copy(tmpFile, resp.Body) // Copy the response body to the temporary file
	if err != nil {
		fmt.Println("Error writing to temporary file:", err)
		return
	}

	// Decompress the Gzip file and save to the final path
	err = gunzipFile(tmpFile.Name(), ripedbPath)
	if err != nil {
		fmt.Println("Error decompressing RIPE database:", err)
		return
	}

	fmt.Println("RIPE database updated successfully.")
}

func gunzipFile(source, destination string) error {
	// Decompresses a Gzip file and saves it to the destination
	dir := filepath.Dir(destination) // Extract directory path
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating directory %s: %v", dir, err)
	}

	file, err := os.Open(source) // Open the Gzip source file
	if err != nil {
		return err
	}
	defer file.Close()

	gz, err := gzip.NewReader(file) // Create a Gzip reader
	if err != nil {
		return err
	}
	defer gz.Close()

	out, err := os.Create(destination) // Create the output file
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, gz) // Copy decompressed content to output file
	return err
}

func whitelistNginx(searchName string) {
	// Generates a whitelist for Nginx based on search criteria
	whitelist := processRIPEdb(searchName, "allow %s;")
	writeSortedFile(whitelist, nginxwhitelistPath)
	runNginxTest()
}

func whitelistTestCookie(searchName string) {
	// Generates a TestCookie whitelist
	whitelist := processRIPEdb(searchName, "%s;")
	writeSortedFile(whitelist, tcwhitelistPath)
	runNginxTest()
}

func processRIPEdb(searchName, format string) []string {
	// Processes the RIPE database file to generate IP ranges based on the search criteria
	file, err := os.Open(ripedbPath)
	if err != nil {
		fmt.Println("Error opening RIPE database:", err)
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var results []string
	var ipRangeStart, ipRangeEnd string
	rangeRegex := regexp.MustCompile(`inetnum:\s+(\S+)\s+-\s+(\S+)`)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(strings.ToLower(line), strings.ToLower(searchName)) {
			for scanner.Scan() {
				line = scanner.Text()
				matches := rangeRegex.FindStringSubmatch(line)
				if len(matches) == 3 {
					ipRangeStart = matches[1]
					ipRangeEnd = matches[2]
					parsedRanges := generateCIDR(ipRangeStart, ipRangeEnd)
					results = append(results, formatRanges(parsedRanges, format)...)
					break
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading RIPE database:", err)
	}
	return results
}

func formatRanges(cidrs []string, format string) []string {
	// Formats a list of CIDR blocks based on a given string format
	var formatted []string
	for _, cidr := range cidrs {
		formatted = append(formatted, fmt.Sprintf(format, cidr))
	}
	return formatted
}

func writeSortedFile(entries []string, path string) {
	// Writes sorted entries to a file
	sort.Strings(entries)
	file, err := os.Create(path)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	for _, entry := range entries {
		file.WriteString(entry + "\n")
	}
}

func runNginxTest() {
	// Runs Nginx configuration test
	cmd := exec.Command("nginx", "-t")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("Nginx configuration test failed.")
	}
}

func blacklistTOR() {
	// Fetches and processes a list of TOR exit addresses for blacklisting
	resp, err := http.Get("https://check.torproject.org/exit-addresses")
	if err != nil {
		fmt.Println("Error fetching TOR addresses:", err)
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	var addresses []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ExitAddress") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				addresses = append(addresses, fmt.Sprintf("deny %s;", fields[1]))
			}
		}
	}

	writeSortedFile(addresses, "tor_blacklist.conf")
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

func uint32ToIP(n uint32) net.IP {
	// Converts a uint32 to a net.IP address.
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip
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

							// Generate CIDR blocks from the IP range.
							cidrList := generateCIDR(ipRangeStart, ipRangeEnd)

							// Log the converted CIDR list.
							for _, cidr := range cidrList {
								fmt.Printf("Converted to CIDR: %s\n", cidr)
							}

							// Add the generated CIDR list to the collection of IP ranges.
							ipRanges = append(ipRanges, cidrList...)
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

	// Create the ACL file for BIND.
	aclFilePath := fmt.Sprintf("/etc/bind/acl_%s.conf", countryCode)
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


func generateCIDR(startIPStr, endIPStr string) []string {
	// Converts a range of IP addresses (startIP to endIP) into a list of CIDR blocks.

	// Parse the start and end IP addresses.
	startIP := net.ParseIP(startIPStr).To4()
	endIP := net.ParseIP(endIPStr).To4()
	if startIP == nil || endIP == nil {
		fmt.Printf("Error: Invalid IP address range: %s - %s\n", startIPStr, endIPStr)
		return nil
	}

	// Convert IP addresses to uint32 for easy arithmetic.
	start := binary.BigEndian.Uint32(startIP)
	end := binary.BigEndian.Uint32(endIP)

	var cidrs []string

	// Loop until the entire range is covered.
	for start <= end {
		// Determine the max size of the CIDR block.
		maxSize := uint32(32)
		for maxSize > 0 {
			// Calculate the network mask.
			mask := uint32(0xFFFFFFFF) << (32 - maxSize)
			// Calculate the masked base address.
			maskedBase := start & mask

			// Check if the base address matches the start address.
			if maskedBase != start {
				break
			}

			// Calculate the broadcast address of this subnet.
			broadcast := maskedBase | (^mask)

			// If the broadcast address is beyond the end IP, reduce the subnet size.
			if broadcast > end {
				break
			}

			// Increase the subnet size.
			maxSize--
		}
		maxSize++

		// Add the CIDR block to the list.
		cidr := fmt.Sprintf("%s/%d", uint32ToIP(start).String(), maxSize)
		cidrs = append(cidrs, cidr)
		fmt.Printf("Converted to CIDR: %s\n", cidr)

		// Increment the start IP to the next block.
		start += 1 << (32 - maxSize)
	}

	return cidrs
}

