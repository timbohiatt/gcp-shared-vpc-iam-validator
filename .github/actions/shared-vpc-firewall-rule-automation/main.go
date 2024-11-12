package main

import (
	"context"
	"fmt"
	"log"
	"os"

	compute "google.golang.org/api/compute/v1"
)

var approvedRoles = map[string]bool { 
	"roles/compute.networkUser": true ,
	"roles/compute.networkAdmin": true ,
}


func main() {

	userEmail, ok := os.LookupEnv("USER_EMAIL")
	if !ok {
		panic("USER_EMAIL environment variable is not set")
	}

	gcpProjectId, ok := os.LookupEnv("GCP_PROJECT_ID")
	if !ok {
		panic("GCP_PROJECT environment variable is not set")
	}

	subnetName, ok := os.LookupEnv("SUBNET_NAME")
	if !ok {
		panic("SUBNET_NAME environment variable is not set")
	}

	subnetRegion, ok := os.LookupEnv("SUBNET_REGION")
	if !ok {
		panic("SUBNET_REGION environment variable is not set")
	}


	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Get the IAM policy for the subnet.
	policy, err := computeService.Subnetworks.GetIamPolicy(gcpProjectId, subnetRegion, subnetName).Context(ctx).Do()
	if err != nil {
		log.Fatal(err)
	}

	// Check if the user has the required permission.
	hasAccess := false
	for _, binding := range policy.Bindings {
		if approvedRoles[binding.Role] {
			for _, member := range binding.Members {
				if fmt.Sprintf("user:%s", userEmail) == member{
					hasAccess = true
					break
				}	
			}		
		}
	}

	if hasAccess {
		fmt.Printf("User: %s has '%s' permission on subnet '%s'\n", userEmail, "roles/compute.networkUser", subnetName)
	} else {
		fmt.Printf("User: %s does not have '%s' permission on subnet '%s'\n", userEmail, "roles/compute.networkUser", subnetName)
	}
}
