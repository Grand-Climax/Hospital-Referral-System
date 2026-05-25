package usecase

import (
	"testing"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		key     string
		value   string
		wantErr bool
	}{
		// Valid keys and values
		{"buffer_days", "2", false},
		{"aging_factor", "1.5", false},
		{"max_horizon_days", "14", false},
		{"overbook_limit_default", "0", false},
		{"auto_notify", "true", false},
		{"sms_otp_enabled", "true", false},
		{"mfa_enabled", "false", false},
		{"mfa_sms_fallback_email", "true", false},
		{"mfa_otp_ttl_seconds", "300", false},
		{"mfa_otp_max_attempts", "5", false},
		{"mfa_otp_resend_cooldown_seconds", "60", false},
		{"enable_cron_jobs", "true", false},
		{"last_waiting_weight_update", "", false},

		// Value validations
		{"buffer_days", "0", true},
		{"buffer_days", "abc", true},
		{"aging_factor", "0", true},
		{"aging_factor", "-1.2", true},
		{"max_horizon_days", "0", true},
		{"overbook_limit_default", "-1", true},
		{"auto_notify", "yes", true},
		{"sms_otp_enabled", "invalid", true},
		{"mfa_otp_ttl_seconds", "45", true}, // must be >= 60
		{"mfa_otp_max_attempts", "0", true},
		{"mfa_otp_resend_cooldown_seconds", "0", true},
		{"enable_cron_jobs", "yes", true},

		// Invalid config keys
		{"invalid_key", "value", true},
		{"garbage", "123", true},
		{"schema_version", "v16", true},
	}

	for _, tt := range tests {
		t.Run(tt.key+"_"+tt.value, func(t *testing.T) {
			err := validateConfig(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsConfigBoolHelpers(t *testing.T) {
	if !isConfigTrue("true") || !isConfigTrue("1") {
		t.Error("isConfigTrue should accept true/1")
	}
	if isConfigTrue("false") || isConfigTrue("") {
		t.Error("isConfigTrue should reject false/empty")
	}
	if !isConfigFalse("false") || !isConfigFalse("0") || !isConfigFalse("") {
		t.Error("isConfigFalse should accept false/0/empty")
	}
	if isConfigFalse("true") {
		t.Error("isConfigFalse should reject true")
	}
}

func TestReadOnlyConfigKeysStripped(t *testing.T) {
	updates := map[string]string{
		"schema_version": "v16",
		"mfa_enabled":    "true",
	}
	for k := range readOnlyConfigKeys {
		delete(updates, k)
	}
	if _, ok := updates["schema_version"]; ok {
		t.Error("schema_version should be stripped before validation")
	}
	if updates["mfa_enabled"] != "true" {
		t.Error("mfa_enabled should remain after stripping read-only keys")
	}
}
