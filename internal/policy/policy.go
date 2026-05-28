package policy

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"sort"
	"strconv"
	"strings"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/tailscale/hujson"
)

type Policy struct {
	Groups map[string][]string `json:"groups"`
	Hosts  map[string]string   `json:"hosts"`
	ACLs   []ACLRule           `json:"acls"`
	Grants []Grant             `json:"grants"`
}

type ACLRule struct {
	Action string   `json:"action"`
	Src    []string `json:"src"`
	Dst    []string `json:"dst"`
	Proto  string   `json:"proto"`
}

type Grant struct {
	Src []string `json:"src"`
	Dst []string `json:"dst"`
	IP  []string `json:"ip"`
}

var supportedPolicySections = map[string]string{
	"acls":          "ACL rules",
	"grants":        "Grants",
	"ssh":           "SSH rules",
	"groups":        "Groups",
	"tagOwners":     "Tag owners",
	"hosts":         "Hosts",
	"ipsets":        "IP sets",
	"postures":      "Posture conditions",
	"nodeAttrs":     "Node attributes",
	"autoApprovers": "Auto approvers",
	"tests":         "Tests",
}

type EdgeOptions struct {
	Perspective string
}

type dstSelector struct {
	Selector string
	Ports    []string
}

type accessAccumulator struct {
	from         string
	to           string
	protocols    map[string]bool
	ports        map[string]bool
	policyRefs   []api.PolicyRef
	perspectives map[string]bool
}

func Parse(raw string) (Policy, error) {
	standard, err := hujson.Standardize([]byte(raw))
	if err != nil {
		return Policy{}, fmt.Errorf("parse policy HuJSON: %w", err)
	}
	var p Policy
	if err := json.Unmarshal(standard, &p); err != nil {
		return Policy{}, fmt.Errorf("decode policy JSON: %w", err)
	}
	if p.Groups == nil {
		p.Groups = map[string][]string{}
	}
	if p.Hosts == nil {
		p.Hosts = map[string]string{}
	}
	return p, nil
}

func EffectiveAccessEdges(raw string, devices []api.Device, options EdgeOptions) ([]api.Edge, error) {
	p, err := Parse(raw)
	if err != nil {
		return nil, err
	}
	return ResolveEffectiveAccess(p, devices, options), nil
}

func StructuredMap(raw string) (api.PolicyMapResponse, error) {
	response := api.PolicyMapResponse{HuJSON: raw}
	standard, err := hujson.Standardize([]byte(raw))
	if err != nil {
		response.ParseError = fmt.Sprintf("parse policy HuJSON: %v", err)
		return response, nil
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(standard, &root); err != nil {
		response.ParseError = fmt.Sprintf("decode policy JSON: %v", err)
		return response, nil
	}

	names := make([]string, 0, len(root))
	for name := range root {
		names = append(names, name)
	}
	sort.Strings(names)

	response.Sections = make([]api.PolicySection, 0, len(names))
	for _, name := range names {
		rawValue := root[name]
		var value any
		if err := json.Unmarshal(rawValue, &value); err != nil {
			value = string(rawValue)
		}
		description, supported := supportedPolicySections[name]
		section := api.PolicySection{
			Name:        name,
			Type:        sectionType(value),
			Supported:   supported,
			Count:       valueCount(value),
			Entries:     policySectionEntries(name, value),
			Raw:         value,
			Description: description,
		}
		if !supported {
			section.Description = "Unsupported section preserved from the policy file."
		}
		response.Sections = append(response.Sections, section)
	}
	return response, nil
}

func AppendACLRule(raw string, rule api.ACLDraft) (string, error) {
	rule.Action = strings.TrimSpace(rule.Action)
	if rule.Action == "" {
		rule.Action = "accept"
	}
	if rule.Action != "accept" {
		return "", fmt.Errorf("only accept ACL rules are supported")
	}
	rule.Src = compactStrings(rule.Src)
	rule.Dst = compactStrings(rule.Dst)
	rule.Proto = strings.TrimSpace(rule.Proto)
	if len(rule.Src) == 0 {
		return "", fmt.Errorf("at least one source selector is required")
	}
	if len(rule.Dst) == 0 {
		return "", fmt.Errorf("at least one destination selector is required")
	}

	root, err := hujson.Parse([]byte(raw))
	if err != nil {
		return "", fmt.Errorf("parse policy HuJSON: %w", err)
	}
	obj, ok := root.Value.(*hujson.Object)
	if !ok {
		return "", fmt.Errorf("policy root must be an object")
	}

	ruleValue, err := hujson.Parse(marshalRule(rule))
	if err != nil {
		return "", err
	}
	ruleValue.BeforeExtra = []byte("\n\t\t")
	ruleValue.AfterExtra = nil

	aclsValue := findObjectMemberValue(obj, "acls")
	if aclsValue == nil {
		array := &hujson.Array{Elements: []hujson.ArrayElement{ruleValue}, AfterExtra: []byte("\n\t")}
		obj.Members = append(obj.Members, hujson.ObjectMember{
			Name:  hujson.Value{BeforeExtra: []byte("\n\t"), Value: hujson.String("acls"), AfterExtra: []byte(" ")},
			Value: hujson.Value{Value: array},
		})
		obj.AfterExtra = []byte("\n")
		return string(root.Pack()), nil
	}

	acls, ok := aclsValue.Value.(*hujson.Array)
	if !ok {
		return "", fmt.Errorf("policy acls field must be an array")
	}
	if len(acls.Elements) == 0 {
		ruleValue.BeforeExtra = []byte("\n\t\t")
		acls.AfterExtra = []byte("\n\t")
	} else if len(acls.AfterExtra) == 0 {
		acls.AfterExtra = []byte("\n\t")
	}
	acls.Elements = append(acls.Elements, ruleValue)
	return string(root.Pack()), nil
}

func ResolveEffectiveAccess(p Policy, devices []api.Device, options EdgeOptions) []api.Edge {
	acc := map[string]*accessAccumulator{}
	for i, rule := range p.ACLs {
		if rule.Action != "" && rule.Action != "accept" {
			continue
		}
		proto := normalizeProto(rule.Proto)
		for _, src := range rule.Src {
			srcDevices := devicesForSelector(src, p, devices)
			if options.Perspective != "" && !selectorIncludesPerspective(src, p, options.Perspective) {
				srcDevices = nil
			}
			for _, dstRaw := range rule.Dst {
				dst := parseDstSelector(dstRaw)
				dstDevices := devicesForSelector(dst.Selector, p, devices)
				for _, from := range srcDevices {
					for _, to := range dstDevices {
						if from.ID == "" || to.ID == "" {
							continue
						}
						key := from.ID + "\x00" + to.ID
						a := acc[key]
						if a == nil {
							a = &accessAccumulator{
								from:         from.ID,
								to:           to.ID,
								protocols:    map[string]bool{},
								ports:        map[string]bool{},
								perspectives: map[string]bool{},
							}
							acc[key] = a
						}
						a.protocols[proto] = true
						for _, port := range dst.Ports {
							a.ports[port] = true
						}
						if options.Perspective != "" {
							a.perspectives[options.Perspective] = true
						}
						a.policyRefs = append(a.policyRefs, api.PolicyRef{
							Section: "acls",
							Index:   i,
							Src:     src,
							Dst:     dstRaw,
						})
					}
				}
			}
		}
	}

	edges := make([]api.Edge, 0, len(acc))
	for _, a := range acc {
		ports := sortedKeys(a.ports)
		protocols := sortedKeys(a.protocols)
		edges = append(edges, api.Edge{
			ID:           edgeID(a.from, a.to, protocols, ports),
			From:         a.from,
			To:           a.to,
			Kind:         api.EdgeKindACL,
			Labels:       edgeLabels(protocols, ports),
			Protocols:    protocols,
			Ports:        ports,
			AccessScope:  classifyScope(protocols, ports),
			PolicyRefs:   a.policyRefs,
			Perspectives: sortedKeys(a.perspectives),
		})
	}
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].ID < edges[j].ID
	})
	return edges
}

func selectorIncludesPerspective(selector string, p Policy, perspective string) bool {
	if selector == perspective || selector == "*" {
		return true
	}
	if selector == "autogroup:member" && perspective != "" {
		return true
	}
	if strings.HasPrefix(selector, "group:") {
		for _, member := range p.Groups[selector] {
			if member == perspective {
				return true
			}
		}
	}
	return false
}

func findObjectMemberValue(obj *hujson.Object, name string) *hujson.Value {
	for i := range obj.Members {
		lit, ok := obj.Members[i].Name.Value.(hujson.Literal)
		if !ok {
			continue
		}
		if lit.String() == name {
			return &obj.Members[i].Value
		}
	}
	return nil
}

func marshalRule(rule api.ACLDraft) []byte {
	payload := map[string]any{
		"action": rule.Action,
		"src":    rule.Src,
		"dst":    rule.Dst,
	}
	if rule.Proto != "" {
		payload["proto"] = rule.Proto
	}
	b, _ := json.Marshal(payload)
	return b
}

func sectionType(value any) string {
	switch value.(type) {
	case []any:
		return "array"
	case map[string]any:
		return "object"
	case nil:
		return "null"
	default:
		return "value"
	}
}

func valueCount(value any) int {
	switch typed := value.(type) {
	case []any:
		return len(typed)
	case map[string]any:
		return len(typed)
	case nil:
		return 0
	default:
		return 1
	}
}

func policySectionEntries(name string, value any) []api.PolicySectionEntry {
	switch name {
	case "acls":
		return aclEntries(value)
	case "grants":
		return grantEntries(value)
	case "groups", "tagOwners", "hosts", "ipsets", "postures":
		return objectEntries(value)
	case "ssh", "nodeAttrs", "tests":
		return ruleArrayEntries(name, value)
	case "autoApprovers":
		return nestedObjectEntries(value)
	default:
		return genericEntries(value)
	}
}

func aclEntries(value any) []api.PolicySectionEntry {
	rules, ok := value.([]any)
	if !ok {
		return genericEntries(value)
	}
	entries := make([]api.PolicySectionEntry, 0, len(rules))
	for i, raw := range rules {
		rule, _ := raw.(map[string]any)
		action := stringValue(rule["action"])
		if action == "" {
			action = "accept"
		}
		src := stringList(rule["src"])
		dst := stringList(rule["dst"])
		proto := stringValue(rule["proto"])
		summary := strings.Join(src, ", ") + " -> " + strings.Join(dst, ", ")
		if proto != "" {
			summary += " (" + proto + ")"
		}
		entries = append(entries, api.PolicySectionEntry{
			Label:     fmt.Sprintf("#%d %s", i+1, action),
			Summary:   summary,
			Selectors: append(append([]string{}, src...), dst...),
			Value:     raw,
		})
	}
	return entries
}

func grantEntries(value any) []api.PolicySectionEntry {
	rules, ok := value.([]any)
	if !ok {
		return genericEntries(value)
	}
	entries := make([]api.PolicySectionEntry, 0, len(rules))
	for i, raw := range rules {
		rule, _ := raw.(map[string]any)
		src := stringList(rule["src"])
		dst := stringList(rule["dst"])
		ip := stringList(rule["ip"])
		summary := strings.Join(src, ", ") + " -> " + strings.Join(dst, ", ")
		if len(ip) > 0 {
			summary += " ip " + strings.Join(ip, ", ")
		}
		entries = append(entries, api.PolicySectionEntry{
			Label:     fmt.Sprintf("#%d grant", i+1),
			Summary:   summary,
			Selectors: append(append(append([]string{}, src...), dst...), ip...),
			Value:     raw,
		})
	}
	return entries
}

func ruleArrayEntries(name string, value any) []api.PolicySectionEntry {
	rules, ok := value.([]any)
	if !ok {
		return genericEntries(value)
	}
	entries := make([]api.PolicySectionEntry, 0, len(rules))
	for i, raw := range rules {
		entries = append(entries, api.PolicySectionEntry{
			Label:   fmt.Sprintf("#%d %s", i+1, name),
			Summary: compactJSON(raw),
			Value:   raw,
		})
	}
	return entries
}

func objectEntries(value any) []api.PolicySectionEntry {
	obj, ok := value.(map[string]any)
	if !ok {
		return genericEntries(value)
	}
	keys := sortedMapKeys(obj)
	entries := make([]api.PolicySectionEntry, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, api.PolicySectionEntry{
			Label:     key,
			Summary:   valueSummary(obj[key]),
			Selectors: append([]string{key}, stringList(obj[key])...),
			Value:     obj[key],
		})
	}
	return entries
}

func nestedObjectEntries(value any) []api.PolicySectionEntry {
	obj, ok := value.(map[string]any)
	if !ok {
		return genericEntries(value)
	}
	var entries []api.PolicySectionEntry
	for _, outer := range sortedMapKeys(obj) {
		inner, ok := obj[outer].(map[string]any)
		if !ok {
			entries = append(entries, api.PolicySectionEntry{Label: outer, Summary: valueSummary(obj[outer]), Value: obj[outer]})
			continue
		}
		for _, key := range sortedMapKeys(inner) {
			entries = append(entries, api.PolicySectionEntry{
				Label:   outer + "." + key,
				Summary: valueSummary(inner[key]),
				Value:   inner[key],
			})
		}
	}
	return entries
}

func genericEntries(value any) []api.PolicySectionEntry {
	switch typed := value.(type) {
	case []any:
		entries := make([]api.PolicySectionEntry, 0, len(typed))
		for i, item := range typed {
			entries = append(entries, api.PolicySectionEntry{
				Label:   fmt.Sprintf("#%d", i+1),
				Summary: valueSummary(item),
				Value:   item,
			})
		}
		return entries
	case map[string]any:
		return objectEntries(typed)
	default:
		if value == nil {
			return nil
		}
		return []api.PolicySectionEntry{{Label: "value", Summary: valueSummary(value), Value: value}}
	}
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func stringList(value any) []string {
	switch typed := value.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	case string:
		return []string{typed}
	default:
		return nil
	}
}

func valueSummary(value any) string {
	if values := stringList(value); len(values) > 0 {
		return strings.Join(values, ", ")
	}
	return compactJSON(value)
}

func compactJSON(value any) string {
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(b)
}

func sortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func devicesForSelector(selector string, p Policy, devices []api.Device) []api.Device {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil
	}
	if selector == "*" || selector == "autogroup:member" {
		return devicesWithOwner(devices)
	}
	if strings.HasPrefix(selector, "group:") {
		return devicesForUsers(p.Groups[selector], devices)
	}
	if strings.HasPrefix(selector, "tag:") {
		return devicesForTag(selector, devices)
	}
	if host, ok := p.Hosts[selector]; ok {
		return devicesForIPSelector(host, devices)
	}
	if strings.Contains(selector, "@") {
		return devicesForUser(selector, devices)
	}
	if isIPSelector(selector) {
		return devicesForIPSelector(selector, devices)
	}
	return nil
}

func devicesWithOwner(devices []api.Device) []api.Device {
	var out []api.Device
	for _, d := range devices {
		if d.Owner != "" {
			out = append(out, d)
		}
	}
	return out
}

func devicesForUsers(users []string, devices []api.Device) []api.Device {
	userSet := map[string]bool{}
	for _, user := range users {
		userSet[user] = true
	}
	var out []api.Device
	for _, d := range devices {
		if userSet[d.Owner] {
			out = append(out, d)
		}
	}
	return out
}

func devicesForUser(user string, devices []api.Device) []api.Device {
	var out []api.Device
	for _, d := range devices {
		if d.Owner == user {
			out = append(out, d)
		}
	}
	return out
}

func devicesForTag(tag string, devices []api.Device) []api.Device {
	var out []api.Device
	for _, d := range devices {
		for _, deviceTag := range d.Tags {
			if deviceTag == tag {
				out = append(out, d)
				break
			}
		}
	}
	return out
}

func devicesForIPSelector(selector string, devices []api.Device) []api.Device {
	prefix, prefixErr := netip.ParsePrefix(selector)
	addr, addrErr := netip.ParseAddr(selector)
	var out []api.Device
	for _, d := range devices {
		if deviceMatchesIP(d, prefix, prefixErr == nil, addr, addrErr == nil) {
			out = append(out, d)
		}
	}
	return out
}

func deviceMatchesIP(d api.Device, prefix netip.Prefix, prefixOK bool, addr netip.Addr, addrOK bool) bool {
	for _, raw := range d.TailscaleIPs {
		ip, err := netip.ParseAddr(raw)
		if err != nil {
			continue
		}
		if addrOK && ip == addr {
			return true
		}
		if prefixOK && prefix.Contains(ip) {
			return true
		}
	}
	for _, raw := range d.RoutedSubnets {
		devicePrefix, err := netip.ParsePrefix(raw)
		if err != nil {
			continue
		}
		if prefixOK && prefixesOverlap(prefix, devicePrefix) {
			return true
		}
		if addrOK && devicePrefix.Contains(addr) {
			return true
		}
	}
	return false
}

func prefixesOverlap(a, b netip.Prefix) bool {
	return a.Contains(b.Addr()) || b.Contains(a.Addr())
}

func isIPSelector(selector string) bool {
	if _, err := netip.ParseAddr(selector); err == nil {
		return true
	}
	if _, err := netip.ParsePrefix(selector); err == nil {
		return true
	}
	return false
}

func parseDstSelector(raw string) dstSelector {
	if raw == "*:*" {
		return dstSelector{Selector: "*", Ports: []string{"*"}}
	}
	idx := strings.LastIndex(raw, ":")
	if idx < 0 {
		return dstSelector{Selector: raw, Ports: []string{"*"}}
	}
	selector := raw[:idx]
	portsRaw := raw[idx+1:]
	if selector == "" {
		selector = raw
		portsRaw = "*"
	}
	return dstSelector{Selector: selector, Ports: parsePorts(portsRaw)}
}

func parsePorts(raw string) []string {
	if raw == "" || raw == "*" {
		return []string{"*"}
	}
	parts := strings.Split(raw, ",")
	ports := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			ports = append(ports, part)
		}
	}
	if len(ports) == 0 {
		return []string{"*"}
	}
	sort.Strings(ports)
	return ports
}

func normalizeProto(proto string) string {
	proto = strings.ToLower(strings.TrimSpace(proto))
	if proto == "" {
		return "tcp"
	}
	return proto
}

func classifyScope(protocols, ports []string) api.AccessScope {
	if len(ports) == 0 {
		return api.AccessScopeNone
	}
	if contains(ports, "*") || contains(ports, "0-65535") {
		return api.AccessScopeBroad
	}
	if onlyTCP(protocols) && len(ports) == 1 && contains(ports, "22") {
		return api.AccessScopeSSH
	}
	if onlyTCP(protocols) && allPortsIn(ports, "80", "443") {
		return api.AccessScopeHTTP
	}
	if len(ports) == 1 {
		if _, err := strconv.Atoi(ports[0]); err == nil {
			return api.AccessScopeLimited
		}
	}
	return api.AccessScopeCustom
}

func onlyTCP(protocols []string) bool {
	return len(protocols) == 1 && protocols[0] == "tcp"
}

func allPortsIn(ports []string, allowed ...string) bool {
	allowedSet := map[string]bool{}
	for _, port := range allowed {
		allowedSet[port] = true
	}
	for _, port := range ports {
		if !allowedSet[port] {
			return false
		}
	}
	return len(ports) > 0
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func edgeLabels(protocols, ports []string) []string {
	if len(protocols) == 0 && len(ports) == 0 {
		return nil
	}
	return []string{strings.Join(protocols, ","), strings.Join(ports, ",")}
}

func edgeID(from, to string, protocols, ports []string) string {
	parts := []string{"acl", from, to, strings.Join(protocols, "_"), strings.Join(ports, "_")}
	for i, part := range parts {
		parts[i] = sanitize(part)
	}
	return strings.Join(parts, ":")
}

func sanitize(value string) string {
	value = strings.ToLower(value)
	value = strings.NewReplacer(" ", "-", "/", "_", ":", "_", "@", "_", ",", "_", "*", "all").Replace(value)
	return value
}
