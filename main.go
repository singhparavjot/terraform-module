package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

const openAIAPIURL = "https://api.openai.com/v1/chat/completions"

var openAIAPIKey string

// Config struct to hold API key from config.json
type Config struct {
	OpenAIAPIKey string `json:"OPENAI_API_KEY"`
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	// Read OpenAI API Key from config.json
	loadAPIKeyFromConfig()

	// Prompt for Azure resources
	fmt.Print("Enter the Azure resources you want to generate (comma-separated): ")
	resourcesInput, _ := reader.ReadString('\n')
	resources := strings.Split(strings.TrimSpace(resourcesInput), ",")

	// Trim spaces from resource names
	for i, res := range resources {
		resources[i] = strings.TrimSpace(res)
	}

	// Create infrastructure directory
	err := os.MkdirAll("infrastructure", os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create infrastructure directory: %v", err)
	}

	// Generate root Terraform files
	generateRootTerraformFiles(resources)

	// Generate modules for each resource
	for _, resource := range resources {
		generateResourceModule(resource)
	}

	fmt.Println("Terraform modules generated successfully.")
}

// Function to load API key from config.json
func loadAPIKeyFromConfig() {
	configFile, err := os.Open("config.json")
	if err != nil {
		log.Fatalf("Failed to open config.json: %v", err)
	}
	defer configFile.Close()

	var config Config
	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(&config); err != nil {
		log.Fatalf("Failed to parse config.json: %v", err)
	}

	openAIAPIKey = strings.TrimSpace(config.OpenAIAPIKey)
	if openAIAPIKey == "" {
		log.Fatalf("OPENAI_API_KEY is missing in config.json")
	}
}

// Function to generate root-level Terraform files
func generateRootTerraformFiles(resources []string) {
	// Generate provider.tf
	providerContent := `provider "azurerm" {
  features {}
}`
	ioutil.WriteFile("infrastructure/provider.tf", []byte(providerContent), 0644)

	// Generate variables.tf
	variablesContent := `variable "location" {
  description = "Azure region"
  type        = string
}`
	ioutil.WriteFile("infrastructure/variables.tf", []byte(variablesContent), 0644)

	// Generate main.tf with module references
	mainContent := ""
	for _, resource := range resources {
		moduleName := fmt.Sprintf("%s_module", strings.ToLower(strings.ReplaceAll(resource, " ", "_")))
		mainContent += fmt.Sprintf(`module "%s" {
  source = "./%s"
}

`, moduleName, moduleName)
	}
	ioutil.WriteFile("infrastructure/main.tf", []byte(mainContent), 0644)
}

// Function to generate Terraform module for a specific resource
func generateResourceModule(resource string) {
	moduleName := fmt.Sprintf("%s_module", strings.ToLower(strings.ReplaceAll(resource, " ", "_")))
	modulePath := fmt.Sprintf("infrastructure/%s", moduleName)

	// Create module directory
	err := os.MkdirAll(modulePath, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create module directory for %s: %v", resource, err)
	}

	// Generate Terraform code using OpenAI API
	terraformCode := getTerraformCode(resource)

	// Write main.tf
	ioutil.WriteFile(fmt.Sprintf("%s/main.tf", modulePath), []byte(terraformCode), 0644)

	// Write variables.tf (empty for now)
	ioutil.WriteFile(fmt.Sprintf("%s/variables.tf", modulePath), []byte(""), 0644)

	// Write output.tf (empty for now)
	ioutil.WriteFile(fmt.Sprintf("%s/output.tf", modulePath), []byte(""), 0644)
}

// Function to get Terraform code for a resource using OpenAI API
func getTerraformCode(resource string) string {
	prompt := fmt.Sprintf(`Generate Terraform code to create an Azure %s. Use best practices and include all required properties.`, resource)

	requestBody, _ := json.Marshal(map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens":        500,
		"temperature":       0.7,
		"top_p":             1.0,
		"n":                 1,
		"stop":              nil,
		"frequency_penalty": 0.0,
		"presence_penalty":  0.0,
	})

	req, err := http.NewRequest("POST", openAIAPIURL, strings.NewReader(string(requestBody)))
	if err != nil {
		log.Fatalf("Failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", openAIAPIKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		log.Fatalf("OpenAI API error: %s", string(bodyBytes))
	}

	var responseData map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseData)
	if err != nil {
		log.Fatalf("Failed to decode OpenAI API response: %v", err)
	}

	// Extract the generated Terraform code
	choices := responseData["choices"].([]interface{})
	message := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	content := message["content"].(string)

	// Clean up the response content
	terraformCode := extractCodeBlock(content)

	return terraformCode
}

// Helper function to extract code blocks from OpenAI response
func extractCodeBlock(content string) string {
	lines := strings.Split(content, "\n")
	codeLines := []string{}
	insideCodeBlock := false

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if insideCodeBlock {
				// End of code block
				break
			} else {
				// Start of code block
				insideCodeBlock = true
				continue
			}
		}
		if insideCodeBlock {
			codeLines = append(codeLines, line)
		}
	}

	return strings.Join(codeLines, "\n")
}
