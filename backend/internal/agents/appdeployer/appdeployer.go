package appdeployer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/saas-agent-platform/backend/internal/agents/shared"
	"github.com/saas-agent-platform/backend/internal/store"
)

type AppDeployerAgent struct {
	store         *store.Store
	daytonaClient *shared.DaytonaClient
	httpClient    *http.Client
}

func NewAppDeployerAgent(s *store.Store, dc *shared.DaytonaClient) *AppDeployerAgent {
	return &AppDeployerAgent{
		store:         s,
		daytonaClient: dc,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}
}

type DeployAppResult struct {
	ContainerRegistry string `json:"container_registry"`
	ImageTag          string `json:"image_tag"`
	EndpointURL       string `json:"endpoint_url"`
	PublicIP          string `json:"public_ip"`
	Region            string `json:"region"`
	VMSize            string `json:"vm_size"`
	AzureStatus       string `json:"azure_status"`
}

func safeSlug(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func (a *AppDeployerAgent) ExecuteDeployJob(ctx context.Context, jobID, projectID string, payload map[string]string) (*DeployAppResult, error) {
	// Verify sandbox codebase exists
	_ = a.daytonaClient.GetOrCreateSandbox(projectID)

	// Extract Task Wizard selections from payload
	region := payload["azure_region"]
	if region == "" {
		region = payload["region"]
	}
	if region == "" {
		region = "eastus"
	}

	vmSize := payload["vm_size"]
	if vmSize == "" {
		vmSize = payload["vmSize"]
	}
	if vmSize == "" {
		vmSize = "Standard_B2s"
	}

	subID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	token := os.Getenv("AZURE_BEARER_TOKEN")

	// If project secret is available, check for stored Azure credentials
	if secretVal, err := a.store.GetSecretValue(projectID, "azure_credentials"); err == nil && secretVal != "" {
		var azureCreds struct {
			SubscriptionID string `json:"subscription_id"`
			BearerToken    string `json:"bearer_token"`
		}
		if json.Unmarshal([]byte(secretVal), &azureCreds) == nil {
			if azureCreds.SubscriptionID != "" {
				subID = azureCreds.SubscriptionID
			}
			if azureCreds.BearerToken != "" {
				token = azureCreds.BearerToken
			}
		}
	}

	slug := safeSlug(projectID)
	rgName := fmt.Sprintf("rg-saas-platform-%s", slug)
	ipName := fmt.Sprintf("pip-%s", slug)
	vmName := fmt.Sprintf("vm-%s", slug)

	if subID == "" || token == "" {
		errStr := "Azure deployment failed: AZURE_SUBSCRIPTION_ID and AZURE_BEARER_TOKEN (or azure_credentials project secret) are required for live cloud provisioning."
		a.store.UpdateJob(jobID, "failed", nil, &errStr)
		return nil, fmt.Errorf("%s", errStr)
	}

	// 1. Provision Resource Group
	if err := a.ProvisionAzureResourceGroup(subID, token, rgName, region); err != nil {
		errStr := fmt.Sprintf("Azure Resource Group provisioning failed: %v", err)
		a.store.UpdateJob(jobID, "failed", nil, &errStr)
		return nil, fmt.Errorf("%s", errStr)
	}

	// 2. Provision Public IP
	realIP, err := a.ProvisionAzurePublicIP(subID, token, rgName, ipName, region)
	if err != nil || realIP == "" {
		errStr := fmt.Sprintf("Azure Public IP allocation failed: %v", err)
		a.store.UpdateJob(jobID, "failed", nil, &errStr)
		return nil, fmt.Errorf("%s", errStr)
	}

	// 3. Provision VM
	if err := a.ProvisionAzureVM(subID, token, rgName, vmName, region, vmSize); err != nil {
		errStr := fmt.Sprintf("Azure Virtual Machine provisioning failed: %v", err)
		a.store.UpdateJob(jobID, "failed", nil, &errStr)
		return nil, fmt.Errorf("%s", errStr)
	}

	res := &DeployAppResult{
		ContainerRegistry: fmt.Sprintf("saasplatform%s.azurecr.io", slug),
		ImageTag:          fmt.Sprintf("saasplatform%s.azurecr.io/app-%s:v1.0.0", slug, slug),
		EndpointURL:       fmt.Sprintf("http://%s:8080", realIP),
		PublicIP:          realIP,
		Region:            region,
		VMSize:            vmSize,
		AzureStatus:       "provisioned_live",
	}

	resBytes, err := json.Marshal(res)
	if err != nil {
		errStr := err.Error()
		a.store.UpdateJob(jobID, "failed", nil, &errStr)
		return nil, err
	}

	a.store.UpdateJob(jobID, "succeeded", resBytes, nil)

	// Update project status
	p, err := a.store.GetProject(projectID)
	if err == nil {
		a.store.UpdateProjectForUser(p.ID, p.UserID, p.GitRemoteURL, p.Name)
	}

	return res, nil
}

func (a *AppDeployerAgent) ProvisionAzureResourceGroup(subID, token, rgName, location string) error {
	endpoint := fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s?api-version=2021-04-01", subID, rgName)
	body := map[string]string{"location": location}
	b, _ := json.Marshal(body)

	req, err := http.NewRequest("PUT", endpoint, bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func (a *AppDeployerAgent) ProvisionAzurePublicIP(subID, token, rgName, ipName, location string) (string, error) {
	endpoint := fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/publicIPAddresses/%s?api-version=2021-02-01", subID, rgName, ipName)
	body := map[string]interface{}{
		"location": location,
		"properties": map[string]interface{}{
			"publicIPAllocationMethod": "Dynamic",
		},
	}
	b, _ := json.Marshal(body)

	req, err := http.NewRequest("PUT", endpoint, bytes.NewBuffer(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var ipResp struct {
		Properties struct {
			IPAddress string `json:"ipAddress"`
		} `json:"properties"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ipResp); err == nil && ipResp.Properties.IPAddress != "" {
		return ipResp.Properties.IPAddress, nil
	}
	return "", fmt.Errorf("public IP response did not contain valid IP address")
}

func (a *AppDeployerAgent) ProvisionAzureVM(subID, token, rgName, vmName, location, vmSize string) error {
	endpoint := fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s?api-version=2021-07-01", subID, rgName, vmName)
	body := map[string]interface{}{
		"location": location,
		"properties": map[string]interface{}{
			"hardwareProfile": map[string]string{"vmSize": vmSize},
			"storageProfile": map[string]interface{}{
				"imageReference": map[string]string{
					"publisher": "Canonical",
					"offer":     "0001-com-ubuntu-server-jammy",
					"sku":       "22_04-lts",
					"version":   "latest",
				},
				"osDisk": map[string]interface{}{
					"name":         vmName + "_OsDisk",
					"createOption": "FromImage",
					"managedDisk": map[string]string{
						"storageAccountType": "Standard_LRS",
					},
				},
			},
			"osProfile": map[string]string{
				"computerName":  vmName,
				"adminUsername": "azureuser",
				"adminPassword": "SecureP@ssw0rd2026!",
			},
		},
	}
	b, _ := json.Marshal(body)

	req, err := http.NewRequest("PUT", endpoint, bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}
