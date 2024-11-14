package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"

	yaml "gopkg.in/yaml.v2"
)

var approvedRoles = map[string]bool{
	"roles/compute.networkUser":  true,
	"roles/compute.networkAdmin": true,
}

// ValidatorConfig ...
type ValidatorConfig struct {
	absolutePath    string
	rulesPath       string
	validateAll     bool
	userEmail       string
	ruleFiles       []string
	changedFileList string
}

type ValidationResults struct {
	results []*ValidationResult
}

func (r *ValidationResults) outputResults() {

	for _, result := range r.results {
		result.outputResult()
		log.Println("\n\n")
	}
}

type ValidationResult struct {
	file             string
	firewallRuleName string
	errors           []string
	status           bool
}

func (r *ValidationResult) outputResult() {
	log.Println("============================================================")
	log.Println("Firewall Rule Name: ", r.firewallRuleName)
	log.Println("Firewall Rule File: ", r.file)
	log.Println("Firewall Rule File Validated: ", r.status)
	if !r.status {
		for _, err := range r.errors {
			log.Println("\t\t - Validation Error: ", err)
		}
	}
	log.Println("============================================================/n")
}

type FirewallRuleFile struct {
	IngressRules map[string]interface{} `yaml:"ingress"`
	EgressRules  map[string]interface{} `yaml:"egress"`
}

func main() {

	// // Get all environment variables
	// envVars := os.Environ()

	// // Iterate over the slice and print each variable
	// for _, envVar := range envVars {
	// 	fmt.Println(envVar)
	// }

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

	config := &ValidatorConfig{
		absolutePath:    absolutePath,
		rulesPath:       rulesPath,
		validateAll:     validateAllBool,
		userEmail:       userEmail,
		changedFileList: changedFileList,
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

	results.outputResults()

	// // Configure Absolute Path
	// filePath, err := filepath.Abs(fmt.Sprintf("%s%s", absolutePath, filepath.Base(changedFileList)))
	// if err != nil {
	// 	panic(err)
	// 	log.Println("Error: Unable to Process the CSV file containing the list of changed firewall definition files.")
	// 	log.Println("Error: CSV File should be located in the Root Directory of your github repository.")
	// 	log.Fatalln("Technical Error: ", err)
	// }

	// // Process CSV
	// err = processCSV(filePath)
	// if err != nil {
	// 	log.Println("Error: Unable to Process the CSV file containing the list of changed firewall definition files.")
	// 	log.Fatalln("Technical Error: ", err)
	// }
}

func calculateRuleFiles(c *ValidatorConfig) (status bool, err error) {
	if c.validateAll {
		// Validate all Firewall Rules listed in all YAML files within the current Git Commit.
		log.Println("Info: loading all firewall rules stored in YAML files in current git commit")
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
			results.results = append(results.results, validateRule(filePath, ruleName, ruleValue))
		}

		// Validate Egress Rules
		for ruleName, ruleValue := range fwRuleFile.EgressRules {
			// Process Each Egress Rule
			results.results = append(results.results, validateRule(filePath, ruleName, ruleValue))
		}

		// validate rule contains destination_range (ingress)
		// validate rule contains source_range (egress)
		// validate rule contains subnet name
		// validate rule contains subnet region
		// validate subnet in region exists
		// get all subnet ip cidrs
		// validate source_range or destination_range in rule within subnet ip cidr
		// validate github actor has roles on subnet
	}

	// Return PASS or FAIL Response.
	return true, results, nil
}

func validateRule(filePath, ruleName string, rule interface{}) *ValidationResult {

	result := &ValidationResult{
		file:             filePath,
		firewallRuleName: ruleName,
		status:           true,
	}

	if ruleWithSubnetName, ok := rule.(map[interface{}]interface{}); ok {

		// Check if Rule has Subnet Name
		if subnetName, ok := ruleWithSubnetName["subnet_name"].(string); ok {
			fmt.Println("Subnet Name:", subnetName)
		} else {
			result.status = false
			result.errors = append(result.errors, "Firewall Rule Configuration Missing Key/Value: subnet_name")
		}

		// Check if Rule has Subnet Region
		if subnetName, ok := ruleWithSubnetName["subnet_region"].(string); ok {
			fmt.Println("Subnet Name:", subnetName)
		} else {
			result.status = false
			result.errors = append(result.errors, "Firewall Rule Configuration Missing Key/Value: subnet_region")
		}
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

// ctx := context.Background()
// computeService, err := compute.NewService(ctx)
// if err != nil {
// 	log.Fatal(err)
// }

// // Get the IAM policy for the subnet.
// policy, err := computeService.Subnetworks.GetIamPolicy(gcpProjectId, subnetRegion, subnetName).Context(ctx).Do()
// if err != nil {
// 	log.Fatal(err)
// }

// // Check if the user has the required permission.
// hasAccess := false
// for _, binding := range policy.Bindings {
// 	if approvedRoles[binding.Role] {
// 		for _, member := range binding.Members {
// 			if fmt.Sprintf("user:%s", userEmail) == member{
// 				hasAccess = true
// 				break
// 			}
// 		}
// 	}
// }

// if hasAccess {
// 	fmt.Printf("User: %s has '%s' permission on subnet '%s'\n", userEmail, "roles/compute.networkUser", subnetName)
// } else {
// 	fmt.Printf("User: %s does not have '%s' permission on subnet '%s'\n", userEmail, "roles/compute.networkUser", subnetName)
// }
