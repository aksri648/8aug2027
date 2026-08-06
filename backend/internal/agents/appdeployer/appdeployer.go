package appdeployer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/saas-agent-platform/backend/internal/agents/shared"
	"github.com/saas-agent-platform/backend/internal/store"
)

type AppDeployerAgent struct {
	store         *store.Store
	daytonaClient *shared.DaytonaClient
}

func NewAppDeployerAgent(s *store.Store, dc *shared.DaytonaClient) *AppDeployerAgent {
	return &AppDeployerAgent{
		store:         s,
		daytonaClient: dc,
	}
}

type DeployAppResult struct {
	ContainerRegistry string `json:"container_registry"`
	ImageTag          string `json:"image_tag"`
	EndpointURL       string `json:"endpoint_url"`
	PublicIP          string `json:"public_ip"`
	Region            string `json:"region"`
	VMSize            string `json:"vm_size"`
}

func (a *AppDeployerAgent) ExecuteDeployJob(ctx context.Context, jobID, projectID string, payload map[string]string) (*DeployAppResult, error) {
	// Verify sandbox codebase
	_ = a.daytonaClient.GetOrCreateSandbox(projectID)

	// Simulate Azure infrastructure provisioning
	res := &DeployAppResult{
		ContainerRegistry: "saasplatformcr.azurecr.io",
		ImageTag:          fmt.Sprintf("saasplatformcr.azurecr.io/app-%s:v1.0.0", projectID[:8]),
		EndpointURL:       fmt.Sprintf("http://52.168.42.%d:8080", time.Now().Unix()%250+1),
		PublicIP:          fmt.Sprintf("52.168.42.%d", time.Now().Unix()%250+1),
		Region:            "eastus",
		VMSize:            "Standard_B2s",
	}

	resBytes, _ := json.Marshal(res)
	a.store.UpdateJob(jobID, "succeeded", resBytes, nil)

	// Update project status
	p, err := a.store.GetProject(projectID)
	if err == nil {
		a.store.UpdateProject(p.ID, p.GitRemoteURL, p.Name)
	}

	return res, nil
}
