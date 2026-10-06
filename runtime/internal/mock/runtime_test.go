package mock

import (
	"context"
	"testing"

	commonv1 "piper_agent/runtime/pkg/pb/common/v1"
	runtimev1 "piper_agent/runtime/pkg/pb/runtime/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestValidateAndRunMock(t *testing.T) {
	rt := NewRuntime(2)
	ctx := context.Background()

	val, err := rt.ValidateTemplate(ctx, &runtimev1.ValidateTemplateRequest{
		RequestId: "r1",
		Template: &commonv1.TemplateDoc{
			Id:          "demo-tpl",
			Name:        "demo",
			JsonPayload: []byte(`{}`),
		},
	})
	if err != nil || !val.GetOk() {
		t.Fatalf("validate: ok=%v err=%v diags=%v", val.GetOk(), err, val.GetDiagnostics())
	}

	run, err := rt.RunTemplate(ctx, &runtimev1.RunTemplateRequest{
		RequestId:  "r2",
		TemplateId: "demo-tpl",
		SessionId:  "sess-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.GetRunId() == "" {
		t.Fatal("expected run_id")
	}
	if run.GetStatus().GetPhase() != commonv1.RunPhase_RUN_PHASE_RUNNING {
		t.Fatalf("expected running, got %v", run.GetStatus().GetPhase())
	}
}

func TestRunRequiresValidate(t *testing.T) {
	rt := NewRuntime(2)
	_, err := rt.RunTemplate(context.Background(), &runtimev1.RunTemplateRequest{
		RequestId:  "r1",
		TemplateId: "unknown",
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", err)
	}
}
