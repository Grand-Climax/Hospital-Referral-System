package dto

import "testing"

func TestResolveStaffIDFromBody(t *testing.T) {
	staffID := "d3000000-0000-0000-0000-000000000003"

	tests := []struct {
		name string
		raw  map[string]interface{}
		want string
	}{
		{name: "staff_id", raw: map[string]interface{}{"staff_id": staffID}, want: staffID},
		{name: "staffId", raw: map[string]interface{}{"staffId": staffID}, want: staffID},
		{name: "id", raw: map[string]interface{}{"id": staffID}, want: staffID},
		{name: "nested staff", raw: map[string]interface{}{"staff": map[string]interface{}{"id": staffID}}, want: staffID},
		{name: "empty", raw: map[string]interface{}{}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveStaffIDFromBody(tt.raw); got != tt.want {
				t.Fatalf("ResolveStaffIDFromBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHospitalAdminSetDepartmentActiveRequest_ResolvedIsActive(t *testing.T) {
	active := true
	req := HospitalAdminSetDepartmentActiveRequest{IsActiveCamel: &active}
	if req.ResolvedIsActive() == nil || !*req.ResolvedIsActive() {
		t.Fatal("expected isActive camelCase to resolve")
	}
}
