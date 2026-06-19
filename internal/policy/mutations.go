package policy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/tailscale/hujson"
)

// ApplyMutation applies a structured edit to a policy HuJSON document.
func ApplyMutation(raw string, mutation api.PolicyMutation) (string, error) {
	switch mutation.Type {
	case "append-acl":
		return AppendACLRule(raw, mutation.Rule)
	case "append-grant":
		return appendGrantRule(raw, mutation.Grant)
	case "remove-acl":
		return removeArrayEntry(raw, "acls", mutation.Index)
	case "upsert-group":
		return upsertObjectEntry(raw, "groups", mutation.Key, mutation.Members)
	case "upsert-tag":
		return upsertObjectEntry(raw, "tagOwners", mutation.Key, mutation.Owners)
	case "upsert-host":
		return upsertHostEntry(raw, mutation.Key, mutation.Host)
	case "upsert-ipset":
		return upsertIPSetEntry(raw, mutation.Key, mutation.IPSet)
	case "upsert-posture":
		return upsertPostureEntry(raw, mutation.Key, mutation.Posture)
	case "append-ssh":
		return appendArrayObject(raw, "ssh", mutation.Value)
	case "upsert-section-json":
		return upsertSectionJSON(raw, mutation.Section, mutation.Value)
	default:
		return "", fmt.Errorf("unsupported mutation type %q", mutation.Type)
	}
}

func appendGrantRule(raw string, grant api.GrantDraft) (string, error) {
	grant.Src = compactStrings(grant.Src)
	grant.Dst = compactStrings(grant.Dst)
	grant.IP = compactStrings(grant.IP)
	grant.SrcPosture = compactStrings(grant.SrcPosture)
	grant.Via = compactStrings(grant.Via)
	if len(grant.Src) == 0 {
		return "", fmt.Errorf("at least one grant source is required")
	}
	if len(grant.Dst) == 0 {
		return "", fmt.Errorf("at least one grant destination is required")
	}
	payload := map[string]any{"src": grant.Src, "dst": grant.Dst}
	if len(grant.IP) > 0 {
		payload["ip"] = grant.IP
	}
	if len(grant.App) > 0 {
		payload["app"] = grant.App
	}
	if len(grant.SrcPosture) > 0 {
		payload["srcPosture"] = grant.SrcPosture
	}
	if len(grant.Via) > 0 {
		payload["via"] = grant.Via
	}
	return appendParsedObject(raw, "grants", payload)
}

func upsertObjectEntry(raw, section, key string, values []string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", fmt.Errorf("%s key is required", section)
	}
	values = compactStrings(values)
	if len(values) == 0 {
		return "", fmt.Errorf("at least one value is required")
	}
	return upsertObjectMapEntry(raw, section, key, values)
}

func upsertHostEntry(raw, key, host string) (string, error) {
	key = strings.TrimSpace(key)
	host = strings.TrimSpace(host)
	if key == "" || host == "" {
		return "", fmt.Errorf("host name and target are required")
	}
	return upsertObjectMapEntry(raw, "hosts", key, host)
}

func upsertIPSetEntry(raw, key string, targets []string) (string, error) {
	key = strings.TrimSpace(key)
	targets = compactStrings(targets)
	if key == "" || len(targets) == 0 {
		return "", fmt.Errorf("IP set name and at least one target are required")
	}
	return upsertObjectMapEntry(raw, "ipsets", key, targets)
}

func upsertPostureEntry(raw, key string, assertions []string) (string, error) {
	key = strings.TrimSpace(key)
	assertions = compactStrings(assertions)
	if key == "" || len(assertions) == 0 {
		return "", fmt.Errorf("posture name and at least one assertion are required")
	}
	if !strings.HasPrefix(key, "posture:") {
		return "", fmt.Errorf("posture name must start with posture:")
	}
	return upsertObjectMapEntry(raw, "postures", key, assertions)
}

func upsertObjectMapEntry(raw, section, key string, value any) (string, error) {
	root, obj, err := parsePolicyObject(raw)
	if err != nil {
		return "", err
	}
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	parsed, err := hujson.Parse(valueBytes)
	if err != nil {
		return "", err
	}
	sectionValue := findObjectMemberValue(obj, section)
	if sectionValue == nil {
		newObj := &hujson.Object{Members: []hujson.ObjectMember{{
			Name:  hujson.Value{Value: hujson.String(key), BeforeExtra: []byte("\n\t\t"), AfterExtra: []byte(" ")},
			Value: parsed,
		}}, AfterExtra: []byte("\n\t")}
		obj.Members = append(obj.Members, hujson.ObjectMember{
			Name:  hujson.Value{BeforeExtra: []byte("\n\t"), Value: hujson.String(section), AfterExtra: []byte(" ")},
			Value: hujson.Value{Value: newObj},
		})
		return string(root.Pack()), nil
	}
	mapObj, ok := sectionValue.Value.(*hujson.Object)
	if !ok {
		return "", fmt.Errorf("policy %s field must be an object", section)
	}
	for i := range mapObj.Members {
		lit, ok := mapObj.Members[i].Name.Value.(hujson.Literal)
		if ok && lit.String() == key {
			mapObj.Members[i].Value = parsed
			return string(root.Pack()), nil
		}
	}
	mapObj.Members = append(mapObj.Members, hujson.ObjectMember{
		Name:  hujson.Value{BeforeExtra: []byte("\n\t\t"), Value: hujson.String(key), AfterExtra: []byte(" ")},
		Value: parsed,
	})
	return string(root.Pack()), nil
}

func appendArrayObject(raw, section string, value json.RawMessage) (string, error) {
	if len(value) == 0 {
		return "", fmt.Errorf("value is required")
	}
	parsed, err := hujson.Parse(value)
	if err != nil {
		return "", fmt.Errorf("parse value: %w", err)
	}
	return appendToArraySection(raw, section, parsed)
}

func upsertSectionJSON(raw, section string, value json.RawMessage) (string, error) {
	section = strings.TrimSpace(section)
	if section == "" {
		return "", fmt.Errorf("section is required")
	}
	if len(value) == 0 {
		return "", fmt.Errorf("value is required")
	}
	root, obj, err := parsePolicyObject(raw)
	if err != nil {
		return "", err
	}
	parsed, err := hujson.Parse(value)
	if err != nil {
		return "", err
	}
	for i := range obj.Members {
		lit, ok := obj.Members[i].Name.Value.(hujson.Literal)
		if ok && lit.String() == section {
			obj.Members[i].Value = parsed
			return string(root.Pack()), nil
		}
	}
	obj.Members = append(obj.Members, hujson.ObjectMember{
		Name:  hujson.Value{BeforeExtra: []byte("\n\t"), Value: hujson.String(section), AfterExtra: []byte(" ")},
		Value: parsed,
	})
	return string(root.Pack()), nil
}

func removeArrayEntry(raw, section string, index int) (string, error) {
	if index < 0 {
		return "", fmt.Errorf("invalid index")
	}
	root, obj, err := parsePolicyObject(raw)
	if err != nil {
		return "", err
	}
	sectionValue := findObjectMemberValue(obj, section)
	if sectionValue == nil {
		return "", fmt.Errorf("section %q not found", section)
	}
	arr, ok := sectionValue.Value.(*hujson.Array)
	if !ok {
		return "", fmt.Errorf("section %q must be an array", section)
	}
	if index >= len(arr.Elements) {
		return "", fmt.Errorf("index out of range")
	}
	arr.Elements = append(arr.Elements[:index], arr.Elements[index+1:]...)
	return string(root.Pack()), nil
}

func appendParsedObject(raw, section string, payload map[string]any) (string, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	parsed, err := hujson.Parse(b)
	if err != nil {
		return "", err
	}
	return appendToArraySection(raw, section, parsed)
}

func appendToArraySection(raw, section string, element hujson.Value) (string, error) {
	root, obj, err := parsePolicyObject(raw)
	if err != nil {
		return "", err
	}
	element.BeforeExtra = []byte("\n\t\t")
	sectionValue := findObjectMemberValue(obj, section)
	if sectionValue == nil {
		array := &hujson.Array{Elements: []hujson.Value{element}, AfterExtra: []byte("\n\t")}
		obj.Members = append(obj.Members, hujson.ObjectMember{
			Name:  hujson.Value{BeforeExtra: []byte("\n\t"), Value: hujson.String(section), AfterExtra: []byte(" ")},
			Value: hujson.Value{Value: array},
		})
		return string(root.Pack()), nil
	}
	arr, ok := sectionValue.Value.(*hujson.Array)
	if !ok {
		return "", fmt.Errorf("policy %s field must be an array", section)
	}
	if len(arr.AfterExtra) == 0 {
		arr.AfterExtra = []byte("\n\t")
	}
	arr.Elements = append(arr.Elements, element)
	return string(root.Pack()), nil
}

func parsePolicyObject(raw string) (hujson.Value, *hujson.Object, error) {
	root, err := hujson.Parse([]byte(raw))
	if err != nil {
		return hujson.Value{}, nil, fmt.Errorf("parse policy HuJSON: %w", err)
	}
	obj, ok := root.Value.(*hujson.Object)
	if !ok {
		return hujson.Value{}, nil, fmt.Errorf("policy root must be an object")
	}
	return root, obj, nil
}
