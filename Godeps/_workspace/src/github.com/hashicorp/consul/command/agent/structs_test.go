package agent

import (
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/hashicorp/consul/consul/structs"
	"testing"
)

func TestAgentStructs_HealthCheck(t *testing.T) {
	def := CheckDefinition{}
	check := def.HealthCheck("node1")

	// Health checks default to critical state
	if check.Status != structs.HealthCritical {
		t.Fatalf("bad: %v", check.Status)
	}
}
