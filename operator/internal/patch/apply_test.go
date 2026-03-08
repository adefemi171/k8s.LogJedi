package patch

import (
	"encoding/json"
	"testing"
)

func TestFilterPatch_nil(t *testing.T) {
	if got := FilterPatch(nil); got != nil {
		t.Errorf("FilterPatch(nil) = %v, want nil", got)
	}
}

func TestFilterPatch_empty(t *testing.T) {
	got := FilterPatch(map[string]interface{}{})
	if got == nil {
		t.Fatal("FilterPatch({}) should not return nil")
	}
	if len(got) != 0 {
		t.Errorf("FilterPatch({}) = %v, want empty map", got)
	}
}

func TestFilterPatch_allowedFields(t *testing.T) {
	in := map[string]interface{}{
		"spec": map[string]interface{}{
			"replicas": float64(3),
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name":  "app",
							"image": "nginx:latest",
							"env":   []interface{}{},
							"resources": map[string]interface{}{
								"limits": map[string]interface{}{"cpu": "500m"},
							},
						},
					},
				},
			},
		},
	}
	got := FilterPatch(in)
	if got == nil {
		t.Fatal("FilterPatch: got nil")
	}
	spec, ok := got["spec"].(map[string]interface{})
	if !ok {
		t.Fatalf("FilterPatch: spec not map, got %T", got["spec"])
	}
	if spec["replicas"] != float64(3) {
		t.Errorf("replicas = %v, want 3", spec["replicas"])
	}
	template, _ := spec["template"].(map[string]interface{})
	if template == nil {
		t.Fatal("template missing")
	}
	podSpec, _ := template["spec"].(map[string]interface{})
	if podSpec == nil {
		t.Fatal("template.spec missing")
	}
	containers, _ := podSpec["containers"].([]interface{})
	if len(containers) != 1 {
		t.Fatalf("containers len = %d, want 1", len(containers))
	}
	c, _ := containers[0].(map[string]interface{})
	if c["name"] != "app" || c["image"] != "nginx:latest" {
		t.Errorf("container = %v", c)
	}
}

func TestFilterPatch_disallowedStripped(t *testing.T) {
	in := map[string]interface{}{
		"spec": map[string]interface{}{
			"replicas": float64(1),
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name":      "app",
							"image":     "nginx:latest",
							"command":   []interface{}{"/bin/sh"},
							"lifecycle": map[string]interface{}{},
						},
					},
				},
			},
		},
	}
	got := FilterPatch(in)
	containers := getContainers(got)
	if len(containers) != 1 {
		t.Fatalf("containers len = %d", len(containers))
	}
	c := containers[0]
	if _, ok := c["command"]; ok {
		t.Error("command should be stripped")
	}
	if _, ok := c["lifecycle"]; ok {
		t.Error("lifecycle should be stripped")
	}
	if c["name"] != "app" || c["image"] != "nginx:latest" {
		t.Errorf("container = %v", c)
	}
}

func getContainers(out map[string]interface{}) []map[string]interface{} {
	spec, _ := out["spec"].(map[string]interface{})
	template, _ := spec["template"].(map[string]interface{})
	podSpec, _ := template["spec"].(map[string]interface{})
	slice, _ := podSpec["containers"].([]interface{})
	var result []map[string]interface{}
	for _, v := range slice {
		if m, ok := v.(map[string]interface{}); ok {
			result = append(result, m)
		}
	}
	return result
}

func TestToJSON(t *testing.T) {
	p := map[string]interface{}{"spec": map[string]interface{}{"replicas": float64(2)}}
	b, err := ToJSON(p)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	spec, _ := decoded["spec"].(map[string]interface{})
	if spec["replicas"] != float64(2) {
		t.Errorf("decoded = %v", decoded)
	}
}
