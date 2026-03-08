package patch

import (
	"encoding/json"
	"strings"
)

// FilterPatch returns a copy of the patch map containing only allowed fields
// (spec.replicas, spec.template.spec.containers with image, env, resources).
func FilterPatch(patch map[string]interface{}) map[string]interface{} {
	if patch == nil {
		return nil
	}
	out := make(map[string]interface{})
	if v, ok := patch["spec"]; ok {
		out["spec"] = filterSpec(v)
	}
	return out
}

func filterSpec(spec interface{}) interface{} {
	m, ok := spec.(map[string]interface{})
	if !ok {
		return nil
	}
	out := make(map[string]interface{})
	if v, ok := m["replicas"]; ok {
		out["replicas"] = v
	}
	if v, ok := m["template"]; ok {
		out["template"] = filterPodTemplate(v)
	}
	return out
}

func filterPodTemplate(template interface{}) interface{} {
	m, ok := template.(map[string]interface{})
	if !ok {
		return nil
	}
	out := make(map[string]interface{})
	if v, ok := m["spec"]; ok {
		out["spec"] = filterPodSpec(v)
	}
	return out
}

func filterPodSpec(spec interface{}) interface{} {
	m, ok := spec.(map[string]interface{})
	if !ok {
		return nil
	}
	out := make(map[string]interface{})
	if v, ok := m["containers"]; ok {
		out["containers"] = filterContainers(v)
	}
	return out
}

func filterContainers(containers interface{}) interface{} {
	slice, ok := containers.([]interface{})
	if !ok {
		return nil
	}
	out := make([]interface{}, 0, len(slice))
	for _, c := range slice {
		if m, ok := c.(map[string]interface{}); ok {
			out = append(out, filterContainer(m))
		}
	}
	return out
}

func filterContainer(c map[string]interface{}) map[string]interface{} {
	allowed := map[string]bool{"name": true, "image": true, "env": true, "resources": true}
	out := make(map[string]interface{})
	for k, v := range c {
		if allowed[strings.ToLower(k)] {
			out[k] = v
		}
	}
	return out
}

// ToJSON returns the patch as JSON bytes for client.RawPatch.
func ToJSON(patch map[string]interface{}) ([]byte, error) {
	return json.Marshal(patch)
}
