package main

import (
	"fmt"
	"log"
	"os"
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

	userEmail, ok := os.LookupEnv("USER_EMAIL")
	if !ok {
		fmt.Println("GitHub Action Error: Required Input 'user-email' not provided.")
		os.Exit(1)
	}

	changedFileList, ok := os.LookupEnv("CHANGED_FILE_LIST")
	if !ok {
		fmt.Println("GitHub Action Error: Required Input 'changed-file-list' not provided.")
		os.Exit(1)
	}

	log.Println("GitHub User Email: " + userEmail)
	log.Println("Changed File List: " + changedFileList)

	_ = userEmail
	_ = changedFileList

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
