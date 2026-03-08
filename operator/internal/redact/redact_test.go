package redact

import (
	"reflect"
	"testing"
)

func TestRedactSpec_envSecret(t *testing.T) {
	spec := map[string]interface{}{
		"containers": []interface{}{
			map[string]interface{}{
				"name": "app",
				"env": []interface{}{
					map[string]interface{}{"name": "NORMAL", "value": "ok"},
					map[string]interface{}{"name": "API_KEY", "value": "secret123"},
					map[string]interface{}{"name": "PASSWORD", "value": "hidden"},
				},
			},
		},
	}
	got, err := RedactSpec(spec)
	if err != nil {
		t.Fatal(err)
	}
	m := got.(map[string]interface{})
	containers := m["containers"].([]interface{})
	c := containers[0].(map[string]interface{})
	env := c["env"].([]interface{})
	if len(env) != 3 {
		t.Fatalf("env len = %d", len(env))
	}
	// NORMAL should remain
	if env[0].(map[string]interface{})["value"] != "ok" {
		t.Errorf("NORMAL value = %v", env[0])
	}
	// API_KEY and PASSWORD should be redacted
	if env[1].(map[string]interface{})["value"] != "REDACTED" {
		t.Errorf("API_KEY value = %v", env[1])
	}
	if env[2].(map[string]interface{})["value"] != "REDACTED" {
		t.Errorf("PASSWORD value = %v", env[2])
	}
}

func TestIsSecretEnvName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"PATH", false},
		{"API_KEY", true},
		{"SECRET_TOKEN", true},
		{"db_password", true},
		{"MY_CREDENTIAL", true},
		{"NORMAL_VAR", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSecretEnvName(tt.name); got != tt.want {
				t.Errorf("isSecretEnvName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestRedactSpec_valueFromRemoved(t *testing.T) {
	spec := map[string]interface{}{
		"containers": []interface{}{
			map[string]interface{}{
				"name": "app",
				"env": []interface{}{
					map[string]interface{}{
						"name": "DB_URL",
						"valueFrom": map[string]interface{}{
							"secretKeyRef": map[string]interface{}{"name": "db-secret", "key": "url"},
						},
					},
				},
			},
		},
	}
	got, err := RedactSpec(spec)
	if err != nil {
		t.Fatal(err)
	}
	containers := got.(map[string]interface{})["containers"].([]interface{})
	env := containers[0].(map[string]interface{})["env"].([]interface{})
	ev := env[0].(map[string]interface{})
	if v, ok := ev["valueFrom"]; ok && v != "REDACTED" {
		// valueFrom should be removed or set to REDACTED
		if reflect.DeepEqual(v, map[string]interface{}{"secretKeyRef": map[string]interface{}{"name": "db-secret", "key": "url"}}) {
			t.Error("valueFrom secretKeyRef should be redacted")
		}
	}
}
