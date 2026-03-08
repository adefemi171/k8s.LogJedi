package redact

import (
	"encoding/json"
	"strings"
)

// SecretEnvSubstrings: env var names containing any of these (case-insensitive) are redacted.
var SecretEnvSubstrings = []string{"SECRET", "TOKEN", "PASSWORD", "KEY", "CREDENTIAL"}

// RedactSpec deep-copies the spec (as a JSON-friendly structure) and redacts
// env var values whose names suggest secrets. Accepts a struct that will be
// marshalled to JSON (e.g. pod/deployment/job spec snippet).
func RedactSpec(spec interface{}) (interface{}, error) {
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	redactMap(m)
	return m, nil
}

func redactMap(m map[string]interface{}) {
	for k, v := range m {
		if v == nil {
			continue
		}
		switch val := v.(type) {
		case map[string]interface{}:
			redactMap(val)
		case []interface{}:
			if k == "containers" || k == "initContainers" {
				for _, item := range val {
					if c, ok := item.(map[string]interface{}); ok {
						redactContainerEnv(c)
					}
				}
			} else if k == "imagePullSecrets" && len(val) > 0 {
				m[k] = []interface{}{map[string]interface{}{"name": "REDACTED"}}
			} else if k == "volumes" {
				for _, item := range val {
					if vm, ok := item.(map[string]interface{}); ok {
						if secret, ok := vm["secret"].(map[string]interface{}); ok {
							delete(secret, "secretName")
							secret["secretName"] = "REDACTED"
						}
					}
				}
				for _, item := range val {
					if sub, ok := item.(map[string]interface{}); ok {
						redactMap(sub)
					}
				}
			} else {
				for _, item := range val {
					if sub, ok := item.(map[string]interface{}); ok {
						redactMap(sub)
					}
				}
			}
		}
	}
}

func redactContainerEnv(container map[string]interface{}) {
	env, ok := container["env"].([]interface{})
	if !ok {
		return
	}
	for _, e := range env {
		ev, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := ev["name"].(string)
		if isSecretEnvName(name) {
			ev["value"] = "REDACTED"
			delete(ev, "valueFrom")
		}
		// Stronger: never send valueFrom (secretKeyRef/configMapKeyRef) to LLM
		if _, hasValueFrom := ev["valueFrom"]; hasValueFrom {
			delete(ev, "valueFrom")
			ev["valueFrom"] = "REDACTED"
		}
	}
}

func isSecretEnvName(name string) bool {
	upper := strings.ToUpper(name)
	for _, sub := range SecretEnvSubstrings {
		if strings.Contains(upper, sub) {
			return true
		}
	}
	return false
}
