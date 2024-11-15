package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"

	compute "google.golang.org/api/compute/v1"

	yaml "gopkg.in/yaml.v2"
)

var approvedRoles = map[string]bool{
	"roles/compute.networkUser":  true,
	"roles/compute.networkAdmin": true,
}

// ValidatorConfig ...
type ValidatorConfig struct {
	hostNetworkProject string
	absolutePath       string
	rulesPath          string
	validateAll        bool
	userEmail          string
	ruleFiles          []string
	changedFileList    string
}

type ValidationResults struct {
	results []*ValidationResult
}

func (r *ValidationResults) pass() bool {
	if len(r.results) > 0 {
		return false
	}
	return true
}

func (r *ValidationResults) outputResults() {
	log.Println()
	log.Println()
	if len(r.results) > 0 {

		log.Println(fmt.Sprintf("Firewall Rules Containing Errors: %d", len(r.results)))
		for _, result := range r.results {
			log.Println()
			log.Println()
			result.outputResult()
		}
	} else {
		log.Println("No Firewall Rules Containing Errors were found.")
		log.Println("Provided Terraform is Valid Firewall Rules Will be Applied.")
	}

}

type ValidationResult struct {
	file             string
	firewallRuleName string
	ruleType         string
	errors           []string
	status           bool
}

func (r *ValidationResult) outputResult() {
	// If Validation Status is not True
	if !r.status {
		log.Println("Firewall Rule:")
		log.Println("  - Name: ", r.firewallRuleName)
		log.Println("  - Source File: ", r.file)
		log.Println("  - Rule Type : ", r.ruleType)
		log.Println("  - Rule Valid? : ", r.status)
		log.Println(fmt.Sprintf("  - Error Count: %d", len(r.errors)))
		log.Println()
		log.Println("Validation Errors:")
		for idx, err := range r.errors {
			// Output Each Validation Error
			log.Println(fmt.Sprintf("\t [%d] Error: %s", idx+1, err))
		}
	}
}

type FirewallRuleFile struct {
	IngressRules map[string]interface{} `yaml:"ingress"`
	EgressRules  map[string]interface{} `yaml:"egress"`
}

func main() {

	absolutePath, ok := os.LookupEnv("ABS_PATH")
	if !ok {
		log.Fatalln("GitHub Action Error: Required Input 'abs-path' not provided.")
	}

	rulesPath, ok := os.LookupEnv("RULES_PATH")
	if !ok {
		log.Fatalln("GitHub Action Error: Required Input 'rules-path' not provided.")
	}

	validateAll, ok := os.LookupEnv("VALIDATE_ALL")
	validateAllBool, err := strconv.ParseBool(validateAll)
	if err != nil {
		log.Fatalln("GitHub Action Error: Required Input 'validate-all' must be 'true' or 'false'. Value: " + validateAll + " is not valid.")
	} else {
		if ok {
			log.Println("Info: Running in Validate Changes Only Mode.")
			log.Println("Info: Only firewall rules that have been modified will be validated.")
		}
		if !ok {
			log.Println("Info: Running in Validate ALL Mode.")
			log.Println("Info: All firewall rules will be validated against the GitHub Actor's User Credentials.")
		}
	}

	userEmail, ok := os.LookupEnv("USER_EMAIL")
	if !ok {
		log.Fatalln("GitHub Action Error: Required Input 'user-email' not provided.")
	}

	changedFileList, ok := os.LookupEnv("CHANGED_FILE_LIST")
	if !ok {
		log.Fatalln("GitHub Action Error: Required Input 'changed-file-list' not provided.")
	}

	hostNetworkProject, ok := os.LookupEnv("GCP_HOST_NETWORK_PROJECT")
	if !ok {
		log.Fatalln("GitHub Action Error: Required Input 'gcp-host-network-project' not provided.")
	}

	config := &ValidatorConfig{
		hostNetworkProject: hostNetworkProject,
		absolutePath:       absolutePath,
		rulesPath:          rulesPath,
		validateAll:        validateAllBool,
		userEmail:          userEmail,
		changedFileList:    changedFileList,
	}

	// Calculate List of YAML Files containing Firewall Rules that need to be processed
	status, err := calculateRuleFiles(config)
	if err != nil {
		// An Error occurred loading, or processing yaml files containing firewall rules for validation
		log.Fatalln("Error: processing firewall rule validation: ", err)
	}

	// No Rule Files were staged or available for processing.
	if !status {
		// Log Error to GitHub workflow and prevent merge.
		log.Fatalln("Error: no firewall rule files or firewall rules to validate.", err)
	}

	// Process the rules
	status, results, err := processRules(config)
	if err != nil {
		log.Fatalln("Error: processing firewall rule validation: ", err)
	}

	// Output Results
	results.outputResults()

	// If Errors Exist Exit the Action
	if !results.pass() {
		log.Fatalln("\n\nError: GitHub Action FAILS Validation")
	}

	return
}

func calculateRuleFiles(c *ValidatorConfig) (status bool, err error) {
	if c.validateAll {
		// Validate all Firewall Rules listed in all YAML files within the current Git branch.
		log.Println("Info: loading all firewall rules stored in YAML files in current branch")
		c.ruleFiles, err = loadAllRulesFiles(c)
		if err != nil {
			return false, fmt.Errorf("Error: reading all firewall rules files: %w", err)
		}
	} else {
		// Validate only Firewall Rules listed in YAML files staged as part of the triggering Git Commit.
		log.Println("Info: loading staged firewall changes based on 'changed file list csv'")
		c.ruleFiles, err = loadStagedRulesFiles(c)
		if err != nil {
			return false, fmt.Errorf("Error: reading all changed firewall rules files: %w", err)
		}
	}

	// At least yaml one file has been selected for validation
	if len(c.ruleFiles) >= 1 {
		return true, nil
	}
	// No yaml files have been selected
	return false, fmt.Errorf("Error: no firewall rule yaml files or firewall rules to validate: %w", err)
}

func processRules(c *ValidatorConfig) (status bool, results ValidationResults, err error) {

	// validate github actor

	for _, filePath := range c.ruleFiles {
		// Load the Firewall Rule File
		fwRuleFile, err := loadFirewallRuleFileToStruct(filePath)
		if err != nil {
			return false, results, fmt.Errorf("Error: reading firewall rules file: %s, %w", filePath, err)
		}

		// Validate Ingress Rules
		for ruleName, ruleValue := range fwRuleFile.IngressRules {
			// Process each Ingress Rule
			result := validateRule(c, "ingress", filePath, ruleName, ruleValue)
			if len(result.errors) >= 1 {
				results.results = append(results.results, result)
			}
		}

		// Validate Egress Rules
		for ruleName, ruleValue := range fwRuleFile.EgressRules {
			// Process Each Egress Rule
			result := validateRule(c, "egress", filePath, ruleName, ruleValue)
			if len(result.errors) >= 1 {
				results.results = append(results.results, result)
			}
		}
	}

	// Return PASS or FAIL Response.
	return true, results, nil
}

func validateRule(c *ValidatorConfig, ruleType, filePath, ruleName string, rule interface{}) *ValidationResult {

	result := &ValidationResult{
		file:             filePath,
		firewallRuleName: ruleName,
		ruleType:         ruleType,
		status:           true,
	}

	if ruleWith, ok := rule.(map[interface{}]interface{}); ok {

		// Declare Values
		var subnetName string
		var subnetRegion string
		var cidrsPendingValidation []string

		// Check if Rule has Subnet Name
		if subnetName, ok = ruleWith["subnet_name"].(string); !ok {
			result.status = false
			result.errors = append(result.errors, "Firewall Rule Configuration Missing Key/Value: subnet_name")
		}

		// Check if Rule has Subnet Region
		if subnetRegion, ok = ruleWith["subnet_region"].(string); !ok {
			result.status = false
			result.errors = append(result.errors, "Firewall Rule Configuration Missing Key/Value: subnet_region")
		}

		// Validations Specific to Ingress Rules
		if ruleType == "ingress" {
			// Assert if destination_ranges is a string array
			if destinationRanges, ok := ruleWith["destination_ranges"].([]interface{}); ok {

				// Check that destination_ranges contains values
				if len(destinationRanges) <= 0 {
					result.status = false
					result.errors = append(result.errors, "Firewall Rule (Egress) Configuration Missing 'destination_ranges' is empty")
				}

				// Collect Up CIDRs for Subnet Validation
				for _, cidr := range destinationRanges {
					cidrsPendingValidation = append(cidrsPendingValidation, cidr.(string))
				}
			} else {
				result.status = false
				result.errors = append(result.errors, "Firewall Rule (Ingress) Configuration Missing Required Key/Value: destination_ranges")
			}
		}

		// Validations Specific to Egress Rules
		if ruleType == "egress" {
			// Assert if source_ranges is a string array
			if destinationRanges, ok := ruleWith["source_ranges"].([]interface{}); ok {

				// Check that source_ranges contains values
				if len(destinationRanges) <= 0 {
					result.status = false
					result.errors = append(result.errors, "Firewall Rule (Egress) Configuration Missing 'source_ranges' is empty")
				}

				// Collect Up CIDRs for Subnet Validation
				for _, cidr := range destinationRanges {
					cidrsPendingValidation = append(cidrsPendingValidation, cidr.(string))
				}
			} else {
				result.status = false
				result.errors = append(result.errors, "Firewall Rule (Egress) Configuration Missing Required Key/Value: source_ranges")
			}
		}

		// No Further Validation Possible Without Subnet Name, Region, Ingress/Egress SRC/DST Ranges
		if !result.status {
			return result
		}

		// Get Subnet IP Ranges & Confirm Subnets existance
		subnetCIDRs, err := getGoogleCloudVPCSubnetCIDRs(c.hostNetworkProject, subnetRegion, subnetName)
		if err != nil {
			result.status = false
			result.errors = append(result.errors, fmt.Sprintf("Firewall Rule (%s) '%s' Configuration Contains Incompatible subnet_name = '%s', subnet_region = '%s' & host_network_project = '%s' values: subnet not found", ruleType, ruleName, subnetName, subnetRegion, c.hostNetworkProject))
			return result
		}

		// Ensure at least One CIDR is Valid
		if len(subnetCIDRs) <= 0 {
			result.status = false
			result.errors = append(result.errors, fmt.Sprintf("Firewall Rule (%s) '%s' Configuration contains a Subnet with no CIDR ranges; Invalid", ruleType, ruleName))
			return result
		}

		// Validate that the Firewall rules contain only source and destination CIDR's that align with the Subnet CIDR's from the configuration
		invalidCIDRs := checkCIDRRanges(cidrsPendingValidation, subnetCIDRs)
		for _, invalidCIDR := range invalidCIDRs {
			result.status = false
			if ruleType == "ingress" {
				result.errors = append(result.errors, fmt.Sprintf("Firewall Rule (%s) '%s' configuration contains a destination_ranges entry: '%s' CIDR that is not part of the firewall rules Primary or Secondary subnet CIDR ranges; Invalid ", ruleType, ruleName, invalidCIDR))
			} else {
				result.errors = append(result.errors, fmt.Sprintf("Firewall Rule (%s) '%s' configuration contains a source_ranges entry: '%s' CIDR that is not part of the firewall rules Primary or Secondary subnet CIDR ranges; Invalid ", ruleType, ruleName, invalidCIDR))
			}
		}

		// Validate the GitHub Actor has permissions to Operate on that Subnet
		access := validateGitHubActorsSubnetIAMBindings(c.userEmail, c.hostNetworkProject, subnetRegion, subnetName)
		if !access {
			result.status = false
			result.errors = append(result.errors, fmt.Sprintf("User: %s does not have Network User or Higher IAM Permissions on Subnet: %s (region: %s) within the Shared VPC at the time of validation", c.userEmail, subnetName, subnetRegion))
			return result
		}
		// Return Now!
		return result

	}

	return result
}

func loadFirewallRuleFileToStruct(filePath string) (*FirewallRuleFile, error) {
	// Read the YAML file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading YAML file: %w", err)
	}

	// Unmarshal the YAML data into the Config struct
	fwRuleFile := &FirewallRuleFile{}
	err = yaml.Unmarshal(data, fwRuleFile)
	if err != nil {
		return fwRuleFile, fmt.Errorf("error unmarshalling YAML: %w", err)
	}

	return fwRuleFile, nil
}

func loadAllRulesFiles(c *ValidatorConfig) (files []string, err error) {

	path := filepath.Join(c.absolutePath, c.rulesPath)
	log.Println(path)

	err = filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && (filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml" || filepath.Ext(path) == ".YML" || filepath.Ext(path) == ".YAML") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return files, err
	}

	return files, err
}

func loadStagedRulesFiles(c *ValidatorConfig) (files []string, err error) {

	// Calculate relative absolute path for changedFileList in relation to GitHub Action
	path, err := getAbsPath(c.absolutePath, c.changedFileList)
	if err != nil {
		return files, fmt.Errorf("Error: 'changed file list csv file' Absolute Path cannot be calculated: %w", err)
	}

	// Check if the file exists
	err = checkFileExists(path)
	if err != nil {
		// Unable to locate the changed files list csv file.
		return files, fmt.Errorf("Error: 'changed file list csv file' doesn't exist: %w", err)
	}

	// Open the Changed File List CSV file
	file, err := os.Open(path)
	if err != nil {
		return files, fmt.Errorf("Error: 'changed file list csv file' cannot be opened: %w", err)
	}

	// File Opened.
	defer file.Close()

	// Create a CSV reader
	reader := csv.NewReader(file)

	// Check if the file is empty
	records, err := reader.Read() // Try to read the first record
	if err == io.EOF {
		// File is empty, error
		return files, fmt.Errorf("Error: 'changed file list csv file' %s is empty, no rules can be process", path)
	} else if err != nil {
		// File can't be read, also error
		return files, fmt.Errorf("Error: reading 'changed file list csv file': %w", err)
	}

	// Process all the Individual filenames
	if len(records) > 0 {
		for _, record := range records {
			fwRuleFilePath, err := filepath.Abs(fmt.Sprintf("%s/%s", c.absolutePath, record))
			if err != nil {
				return files, fmt.Errorf("Error: 'firewall rule yaml file': %s Absolute Path cannot be calculated: %w", record, err)
			}
			log.Println("Info: Modified Firewall Rule File: ", fwRuleFilePath, " will be validated.")
			files = append(files, fwRuleFilePath)
		}
	} else {
		return files, fmt.Errorf("Error: 'changed file list csv file' %s is empty, no rules can be process", path)
	}

	return files, err
}

// Configure Absolute Path
func getAbsPath(path, filename string) (string, error) {
	outputPath, err := filepath.Abs(fmt.Sprintf("%s/%s", path, filepath.Base(filename)))
	if err != nil {
		return "", err
	}
	return outputPath, nil
}

func checkFileExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("File %s does not exist", path)
	}
	return nil
}

// gcpProjectId, ok := os.LookupEnv("GCP_PROJECT_ID")
// if !ok {
// 	panic("GCP_PROJECT environment variable is not set")
// }

// subnetName, ok := os.LookupEnv("SUBNET_NAME")
// if !ok {
// 	panic("SUBNET_NAME environment variable is not set")
// }

// subnetRegion, ok := os.LookupEnv("SUBNET_REGION")
// if !ok {
// 	panic("SUBNET_REGION environment variable is not set")
// }

//fmt.Sprintln("Acting User: %s", userEmail)

/*

package main

import (
	"fmt"
	"net"
)





func main() {
	// Example usage
	listA1 := []string{"192.168.0.14/32", "192.168.0.15/32"}
	listB1 := []string{"192.168.0.0/29", "192.168.0.8/29"}
	fmt.Println(checkCIDRRanges(listA1, listB1)) // Output: []

	listA2 := []string{"192.168.0.14/32", "10.10.1.0/16"}
	listB2 := []string{"192.168.0.0/29", "192.168.0.8/29"}
	fmt.Println(checkCIDRRanges(listA2, listB2)) // Output: [10.10.1.0/16]

	listA3 := []string{"0.0.0.0/0", "10.10.1.0/16"}
	listB3 := []string{"192.168.0.0/29", "192.168.0.8/29"}
	fmt.Println(checkCIDRRanges(listA3, listB3)) // Output: [0.0.0.0/0 10.10.1.0/16]
}



*/

func getGoogleCloudVPCSubnetCIDRs(projectName string, region string, subnetName string) ([]string, error) {
	// Create a new compute client.
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		// Unable to create a GCP Compute Service
		log.Println(err)
		return nil, err
	}

	// Lookup the subnet in Google Cloud
	subnet, err := computeService.Subnetworks.Get(projectName, region, subnetName).Do()
	if err != nil {
		// Unable to locate the subnet
		log.Println(err)
		return nil, err
	}

	// Define an Empty List os CIDRs
	var cidrRanges = []string{}

	// Collect and store the subnet's primary CIDR range
	if subnet.IpCidrRange != "" {
		cidrRanges = append(cidrRanges, subnet.IpCidrRange)
	}

	// Collect and store the subnet's secondary CIDR ranges
	for _, secondaryRange := range subnet.SecondaryIpRanges {
		cidrRanges = append(cidrRanges, secondaryRange.IpCidrRange)
	}
	return cidrRanges, nil
}

func validateGitHubActorsSubnetIAMBindings(userEmail string, projectName string, region string, subnetName string) bool {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Get the IAM policy for the subnet.
	policy, err := computeService.Subnetworks.GetIamPolicy(projectName, region, subnetName).Context(ctx).Do()
	if err != nil {
		log.Fatal(err)
	}

	// Check if the user has the required permission.
	hasAccess := false
	for _, binding := range policy.Bindings {
		if approvedRoles[binding.Role] {
			for _, member := range binding.Members {
				if fmt.Sprintf("user:%s", userEmail) == member {
					hasAccess = true
					break
				}
			}
		}
	}

	if hasAccess {
		return true
	}
	return false
}

// lastIP calculates the last IP in a given IP network.
func lastIP(n *net.IPNet) net.IP {
	ip := make(net.IP, len(n.IP.To4()))
	copy(ip, n.IP.To4())
	for i := 0; i < len(ip); i++ {
		ip[i] |= ^n.Mask[i]
	}
	return ip
}

func checkCIDRRanges(ruleCIDRs, subnetCIDRs []string) []string {
	failedRanges := []string{}

	for _, cidrA := range ruleCIDRs {
		_, ipNetA, err := net.ParseCIDR(cidrA)
		if err != nil {
			// Handle parsing error if necessary
			fmt.Printf("Error parsing CIDR %s: %v\n", cidrA, err)
			continue
		}

		found := false
		for _, cidrB := range subnetCIDRs {
			_, ipNetB, err := net.ParseCIDR(cidrB)
			if err != nil {
				// Handle parsing error if necessary
				fmt.Printf("Error parsing CIDR %s: %v\n", cidrB, err)
				continue
			}

			if ipNetB.Contains(ipNetA.IP) && ipNetB.Contains(lastIP(ipNetA)) {
				found = true
				break
			}
		}

		if !found {
			failedRanges = append(failedRanges, cidrA)
		}
	}

	return failedRanges
}
