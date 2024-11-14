package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

var approvedRoles = map[string]bool{
	"roles/compute.networkUser":  true,
	"roles/compute.networkAdmin": true,
}

func main() {

	// Get all environment variables
	//envVars := os.Environ()

	// Iterate over the slice and print each variable
	/*for _, envVar := range envVars {
		fmt.Println(envVar)
	}*/

	absolutePath, ok := os.LookupEnv("ABS_PATH")
	if !ok {
		log.Fatalln("GitHub Action Error: Required Input 'abs-path' not provided.")
	}

	validateAll, ok := os.LookupEnv("VALIDATE_ALL")
	if !ok {
		log.Println("GitHub Action Info: Running in Validate Changes Only Mode.")
		log.Println("GitHub Action Info: Only firewall rules that have been modified will be validated.")
	}
	if ok {
		log.Println("GitHub Action Info: Running in Validate ALL Mode.")
		log.Println("GitHub Action Info: All firewall rules will be validated against the GitHub Actor's User Credentials.")
	}

	userEmail, ok := os.LookupEnv("USER_EMAIL")
	if !ok {
		log.Fatalln("GitHub Action Error: Required Input 'user-email' not provided.")
	}

	changedFileList, ok := os.LookupEnv("CHANGED_FILE_LIST")
	if !ok {
		log.Fatalln("GitHub Action Error: Required Input 'changed-file-list' not provided.")
	}

	// Log Out All Variables for Execution Run.
	log.Println("Absolute Path: " + absolutePath)
	log.Println("Validate All: " + validateAll)
	log.Println("GitHub User Email: " + userEmail)
	log.Println("Changed File List: " + changedFileList)

	// Configure Absolute Path
	filePath, err := filepath.Abs(fmt.Sprintf("%s%s", absolutePath, filepath.Base(changedFileList)))
	if err != nil {
		panic(err)
		log.Println("Error: Unable to Process the CSV file containing the list of changed firewall definition files.")
		log.Println("Error: CSV File should be located in the Root Directory of your github repository.")
		log.Fatalln("Technical Error: ", err)
	}

	// Process CSV
	err = processCSV(filePath)
	if err != nil {
		log.Println("Error: Unable to Process the CSV file containing the list of changed firewall definition files.")
		log.Fatalln("Technical Error: ", err)
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
}

func processCSV(filename string) error {
	// Check if the file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("Changed File List CSV file %s does not exist", filename)
	}

	// Open the CSV file
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening file Changed File List CSV file: %w", err)
	}
	defer file.Close()

	// Create a CSV reader
	reader := csv.NewReader(file)

	// Check if the file is empty
	_, err = reader.Read() // Try to read the first record
	if err == io.EOF {
		return fmt.Errorf("Changed File List CSV file %s is empty, no rules to process", filename)
	} else if err != nil {
		return fmt.Errorf("error reading Changed File List CSV file: %w", err)
	}

	// Reset the reader to the beginning of the file
	_, err = file.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("error seeking Changed File List CSV file: %w", err)
	}
	reader = csv.NewReader(file)

	// Read and output each entry on a new line
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break // End of file
		}
		if err != nil {
			return fmt.Errorf("error reading Changed File List CSV file record: %w", err)
		}

		fmt.Println(record)
	}

	return nil
}
