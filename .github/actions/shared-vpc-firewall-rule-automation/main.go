package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
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

func main() {

	// Get all environment variables
	envVars := os.Environ()

	// Iterate over the slice and print each variable
	for _, envVar := range envVars {
		fmt.Println(envVar)
	}

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
		if !ok {
			log.Println("GitHub Action Info: Running in Validate Changes Only Mode.")
			log.Println("GitHub Action Info: Only firewall rules that have been modified will be validated.")
		}
		if ok {
			log.Println("GitHub Action Info: Running in Validate ALL Mode.")
			log.Println("GitHub Action Info: All firewall rules will be validated against the GitHub Actor's User Credentials.")
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

	status, err := processRules(config)
	if err != nil {
		log.Fatalln("Error: processing firewall rule validation: %w", err)
	}

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

func processRules(c *ValidatorConfig) (status bool, err error) {
	if c.validateAll {
		// Validate all Firewall Rules listed in all YAML files within the current Git Commit.
		c.ruleFiles, err = loadAllRulesFiles(c)
		if err != nil {
			return false, fmt.Errorf("Error: reading all firewall rules files: %w", err)
		}
	} else {
		// Validate only Firewall Rules listed in YAML files staged as part of the triggering Git Commit.
		c.ruleFiles, err = loadStagedRulesFiles(c)
		if err != nil {
			return false, fmt.Errorf("Error: reading all changed firewall rules files: %w", err)
		}
	}

	// Process each YAML File and Validate the rules

	// Return PASS or FAIL Response.
	return true, nil
}

func loadAllRulesFiles(c *ValidatorConfig) (files []string, err error) {
	return files, err
}

func loadStagedRulesFiles(c *ValidatorConfig) (files []string, err error) {

	// Calculate relative absolute path for changedFileList in relation to GitHub Action
	path, err := getAbsPath(c.rulesPath, c.changedFileList)
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
	_, err = reader.Read() // Try to read the first record
	if err == io.EOF {
		// File is empty, error
		return files, fmt.Errorf("Error: 'changed file list csv file' %s is empty, no rules can be process", path)
	} else if err != nil {
		// File can't be read, also error
		return files, fmt.Errorf("Error: reading 'changed file list csv file': %w", err)
	}

	// Reset new reader to the restart at the beginning of the file
	reader = csv.NewReader(file)

	// Read and output each entry on a new line
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break // End of file
		}
		if err != nil {
			return files, fmt.Errorf("Error: reading 'changed file list csv file' records: %w", err)
		}

		log.Println(record)
		files = append(files, record[0])
	}

	return files, err
}

// Configure Absolute Path
func getAbsPath(path, filename string) (string, error) {
	outputPath, err := filepath.Abs(fmt.Sprintf("%s%s", path, filepath.Base(filename)))
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
