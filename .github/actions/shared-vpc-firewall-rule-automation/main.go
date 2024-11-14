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

type ValidationResult struct {
	file             string
	firewallRuleName string
	error            string
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

	_ = results
	_ = status

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

func processRules(c *ValidatorConfig) (status bool, results []*ValidationResult, err error) {

	// validate github actor

	for _, filePath := range c.ruleFiles {
		// Load the Firewall Rule File
		fwRuleFile, err := loadFirewallRuleFileToStruct(filePath)
		if err != nil {
			return false, results, fmt.Errorf("Error: reading firewall rules file: %s, %w", filePath, err)
		}

		log.Println(fwRuleFile)

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

			path, err := getAbsPath(c.absolutePath, record)
			if err != nil {
				return files, fmt.Errorf("Error: 'firewall rule yaml file': %s Absolute Path cannot be calculated: %w", record, err)
			}
			log.Println("Info: Modified Firewall Rule File: ", path, " will be validated.")
			files = append(files, path)
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
