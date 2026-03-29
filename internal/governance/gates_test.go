package governance

import "testing"

func TestAssessFlagsRiskyTasks(t *testing.T) {
	assessment := Assess("deploy migration and push secrets to production")
	if !assessment.RequiresApproval {
		t.Fatal("expected approval to be required")
	}
	if assessment.RiskLevel != "high" {
		t.Fatalf("expected high risk, got %s", assessment.RiskLevel)
	}
	if len(assessment.Gates) == 0 {
		t.Fatal("expected gates to be populated")
	}
}
