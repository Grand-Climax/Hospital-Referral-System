package dto

import (
	"net/url"
	"testing"
)

func TestResolveLinkDepartmentFromBody(t *testing.T) {
	deptID := "b5000000-0000-0000-0000-000000000005"

	tests := []struct {
		name           string
		raw            map[string]interface{}
		wantDeptID     string
		wantDailyLimit int
	}{
		{name: "department_id", raw: map[string]interface{}{"department_id": deptID, "daily_limit": 30}, wantDeptID: deptID, wantDailyLimit: 30},
		{name: "departmentId", raw: map[string]interface{}{"departmentId": deptID, "dailyLimit": 25}, wantDeptID: deptID, wantDailyLimit: 25},
		{name: "id", raw: map[string]interface{}{"id": deptID}, wantDeptID: deptID},
		{name: "nested department", raw: map[string]interface{}{"department": map[string]interface{}{"id": deptID}}, wantDeptID: deptID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDeptID, gotDailyLimit := ResolveLinkDepartmentFromBody(tt.raw)
			if gotDeptID != tt.wantDeptID || gotDailyLimit != tt.wantDailyLimit {
				t.Fatalf("ResolveLinkDepartmentFromBody() = (%q, %d), want (%q, %d)", gotDeptID, gotDailyLimit, tt.wantDeptID, tt.wantDailyLimit)
			}
		})
	}
}

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
		{name: "staff string", raw: map[string]interface{}{"staff": staffID}, want: staffID},
		{name: "head string", raw: map[string]interface{}{"head": staffID}, want: staffID},
		{name: "departmentHeadId", raw: map[string]interface{}{"departmentHeadId": staffID}, want: staffID},
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

func TestResolveStaffID(t *testing.T) {
	staffID := "d3000000-0000-0000-0000-000000000003"

	t.Run("bare json string", func(t *testing.T) {
		got := ResolveStaffID([]byte(`"`+staffID+`"`), nil)
		if got != staffID {
			t.Fatalf("ResolveStaffID() = %q, want %q", got, staffID)
		}
	})

	t.Run("query param", func(t *testing.T) {
		q := url.Values{}
		q.Set("staff_id", staffID)
		got := ResolveStaffID(nil, q)
		if got != staffID {
			t.Fatalf("ResolveStaffID() = %q, want %q", got, staffID)
		}
	})
}

func TestHospitalAdminSetDepartmentActiveRequest_ResolvedIsActive(t *testing.T) {
	active := true
	req := HospitalAdminSetDepartmentActiveRequest{IsActiveCamel: &active}
	if req.ResolvedIsActive() == nil || !*req.ResolvedIsActive() {
		t.Fatal("expected isActive camelCase to resolve")
	}
}
