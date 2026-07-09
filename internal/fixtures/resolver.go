package fixtures

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var legacyResponsesBasePath = "internal/webhook/jsons"

var fixtureHTTPClient = &http.Client{Timeout: 10 * time.Second}

var knownSectors = []string{
	"trade", "logistics", "mobility", "hospitality", "finance",
	"commerce", "deg", "energy", "retail",
}

var crcNameToCode = map[string]string{
	"fashion & accessories":            "trc-fashion",
	"electronics & gadgets":            "trc-electronics",
	"food & beverages":                 "trc-food-bev",
	"health & beauty":                  "trc-health-beauty",
	"home & garden":                    "trc-home-garden",
	"furniture & living":               "trc-furniture",
	"automotive parts & accessories":   "trc-automotive",
	"sports & outdoors":                "trc-sports",
	"toys & baby products":             "trc-toys-baby",
	"books & media":                    "trc-media",
	"digital goods":                    "trc-digital",
	"industrial & b2b supplies":        "trc-industrial",
	"arts, crafts & collectibles":      "trc-arts",
	"religious & cultural goods":       "trc-religious",
	"luggage & travel accessories":     "trc-luggage",
}

// PATTERN_NAME_TO_CODE maps the ION spec's real pattern names (flows/trade/patterns/<name>) to
// this app's internal folder codes, so requests shaped like real ION traffic (which send e.g.
// "storefront", not "B2C-SF") still resolve to the right fixtures.
var patternNameToCode = map[string]string{
	"storefront":              "B2C-SF",
	"made-to-order":           "B2C-MTO",
	"subscription":            "B2C-SUB",
	"live-commerce":           "B2C-LIVE",
	"digital-goods":           "B2C-DIG",
	"business-procurement":    "B2B-PP",
	"marketplace-inhouse":     "MP-IH",
	"marketplace-listed":      "MP-IL",
	"forward-auction":         "AUC-F",
	"reverse-auction":         "AUC-R",
	"cross-border":            "XB",
	"government":              "B2G",
}

var safeSegmentRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
var crcCodeRe = regexp.MustCompile(`^trc-[a-z0-9-]+$`)

type responseRoute struct {
	sector  string
	pattern string
	crc     string
}

func getStructuredResponsesBaseURL() string {
	return strings.TrimSpace(os.Getenv("RESPONSE_FIXTURES_BASE_URL"))
}

// NormalizeDomain replaces colons with dots for Windows-compatible dir names, then strips
// the version suffix (e.g. "beckn:ion:retail:1.0" -> "beckn.ion.retail").
func NormalizeDomain(domain string) string {
	if domain == "" {
		return domain
	}
	replaced := strings.ReplaceAll(domain, ":", ".")
	versionSuffixRe := regexp.MustCompile(`\.\d+(?:\.\d+)*$`)
	return versionSuffixRe.ReplaceAllString(replaced, "")
}

// ResolveDomain resolves the domain identifier from a beckn context object, falling back to
// networkId if domain is absent.
func ResolveDomain(context map[string]interface{}) string {
	if context == nil {
		return ""
	}
	if v, ok := firstString(context["domain"]); ok {
		return v
	}
	if v, ok := firstString(context["network_id"]); ok {
		return v
	}
	if v, ok := firstString(context["networkId"]); ok {
		return v
	}
	return ""
}

func headerValue(headers http.Header, name string) string {
	if headers == nil {
		return ""
	}
	return headers.Get(name)
}

func firstString(values ...interface{}) (string, bool) {
	for _, v := range values {
		s, ok := v.(string)
		if ok {
			trimmed := strings.TrimSpace(s)
			if trimmed != "" {
				return trimmed, true
			}
		}
	}
	return "", false
}

func firstStringVals(values ...string) (string, bool) {
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			return trimmed, true
		}
	}
	return "", false
}

func safeSegment(value string, lowercase bool) string {
	if value == "" {
		return ""
	}
	normalized := strings.TrimSpace(value)
	if lowercase {
		normalized = strings.ToLower(normalized)
	}
	if !safeSegmentRe.MatchString(normalized) {
		return ""
	}
	return normalized
}

func sectorFromDomain(domain string) string {
	if domain == "" {
		return ""
	}
	replacer := strings.NewReplacer(":", ".", "/", ".")
	normalized := strings.ToLower(replacer.Replace(domain))
	parts := strings.Split(normalized, ".")
	partSet := make(map[string]bool, len(parts))
	for _, p := range parts {
		if p != "" {
			partSet[p] = true
		}
	}
	for _, sector := range knownSectors {
		if partSet[sector] {
			return sector
		}
	}
	return ""
}

func resolvePatternCode(rawPattern string) string {
	if rawPattern == "" {
		return ""
	}
	key := strings.ToLower(strings.TrimSpace(rawPattern))
	if code, ok := patternNameToCode[key]; ok {
		return code
	}
	return rawPattern
}

func findFirstCrcCode(value interface{}, depth int) string {
	if value == nil || depth > 8 {
		return ""
	}

	switch v := value.(type) {
	case string:
		normalized := strings.ToLower(strings.TrimSpace(v))
		if crcCodeRe.MatchString(normalized) {
			return normalized
		}
		return crcNameToCode[normalized]
	case []interface{}:
		for _, item := range v {
			if found := findFirstCrcCode(item, depth+1); found != "" {
				return found
			}
		}
	case map[string]interface{}:
		for _, item := range v {
			if found := findFirstCrcCode(item, depth+1); found != "" {
				return found
			}
		}
	}
	return ""
}

func asString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func asMap(v interface{}) map[string]interface{} {
	m, _ := v.(map[string]interface{})
	return m
}

func resolveResponseRoute(requestBody map[string]interface{}, headers http.Header) responseRoute {
	meta := asMap(requestBody["_meta"])
	context := asMap(requestBody["context"])

	sector, _ := firstStringVals(
		headerValue(headers, "x-ion-sector"),
		asString(meta["sector"]),
		asString(meta["sectorCode"]),
		asString(context["sector"]),
		sectorFromDomain(asString(context["domain"])),
		sectorFromDomain(asString(context["network_id"])),
		sectorFromDomain(asString(context["networkId"])),
	)
	sector = safeSegment(sector, true)

	rawPattern, _ := firstStringVals(
		headerValue(headers, "x-ion-pattern"),
		asString(meta["pattern"]),
		asString(meta["patternCode"]),
		asString(meta["variant"]),
		asString(context["pattern"]),
		asString(context["patternCode"]),
	)
	pattern := safeSegment(resolvePatternCode(rawPattern), false)

	crc, _ := firstStringVals(
		headerValue(headers, "x-ion-crc"),
		headerValue(headers, "x-ion-resource"),
		asString(meta["crc"]),
		asString(meta["crcCode"]),
		asString(meta["resource"]),
		asString(meta["resourceCode"]),
		asString(context["crc"]),
		asString(context["crcCode"]),
		findFirstCrcCode(requestBody, 0),
	)
	crc = safeSegment(crc, true)

	return responseRoute{sector: sector, pattern: pattern, crc: crc}
}

func unique(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

func patternVariants(pattern string) []string {
	if pattern == "" {
		return nil
	}
	return unique([]string{pattern, strings.ToUpper(pattern)})
}

func pushCandidate(candidates [][]string, segments []string) [][]string {
	for _, segment := range segments {
		if safeSegment(segment, false) == "" {
			return candidates
		}
	}
	return append(candidates, segments)
}

func candidateKey(segments []string) string {
	return strings.Join(segments, "/")
}

func uniqueCandidates(values [][]string) [][]string {
	seen := make(map[string]bool, len(values))
	result := make([][]string, 0, len(values))
	for _, v := range values {
		key := candidateKey(v)
		if !seen[key] {
			seen[key] = true
			result = append(result, v)
		}
	}
	return result
}

func buildStructuredCandidates(route responseRoute, action string, persona string) [][]string {
	var candidates [][]string
	fileName := fmt.Sprintf("%s.json", action)
	sector := route.sector
	crc := route.crc
	safePersona := safeSegment(persona, false)

	if sector == "" {
		return candidates
	}

	for _, pattern := range patternVariants(route.pattern) {
		if crc != "" && safePersona != "" {
			candidates = pushCandidate(candidates, []string{sector, pattern, crc, safePersona, fileName})
		}
		if safePersona != "" {
			candidates = pushCandidate(candidates, []string{sector, pattern, safePersona, fileName})
		}
	}

	if safePersona != "" {
		candidates = pushCandidate(candidates, []string{sector, safePersona, fileName})
	}

	for _, pattern := range patternVariants(route.pattern) {
		if crc != "" {
			candidates = pushCandidate(candidates, []string{sector, pattern, crc, fileName})
		}
		candidates = pushCandidate(candidates, []string{sector, pattern, fileName})
	}

	candidates = pushCandidate(candidates, []string{sector, fileName})

	return uniqueCandidates(candidates)
}

func buildFixtureURL(baseURL string, segments []string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.Join(segments, "/")
}

func fetchRemoteFixture(fullURL string) (interface{}, bool, error) {
	resp, err := fixtureHTTPClient.Get(fullURL)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("unexpected status %d fetching %s", resp.StatusCode, fullURL)
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, false, err
	}
	return result, true, nil
}

func readJSONFile(targetPath string) (interface{}, error) {
	data, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, err
	}
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func stripInternalMeta(payload interface{}) interface{} {
	m, ok := payload.(map[string]interface{})
	if !ok {
		return payload
	}
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		if k == "_meta" {
			continue
		}
		result[k] = v
	}
	return result
}

// ReadRequestResponse resolves and reads the response fixture for the given action, first
// trying structured candidates derived from headers/_meta/context, then falling back to the
// legacy domain-tree lookup.
func ReadRequestResponse(requestBody map[string]interface{}, action string, persona string, headers http.Header) interface{} {
	baseURL := getStructuredResponsesBaseURL()
	route := resolveResponseRoute(requestBody, headers)
	candidates := buildStructuredCandidates(route, action, persona)

	if route.sector == "" {
		slog.Warn("no sector signal found for action, falling back to legacy domain lookup", "action", action)
	}

	if baseURL == "" {
		slog.Warn("RESPONSE_FIXTURES_BASE_URL not set; skipping structured fixture lookup")
	} else {
		for _, segments := range candidates {
			fullURL := buildFixtureURL(baseURL, segments)
			payload, found, err := fetchRemoteFixture(fullURL)
			if err != nil {
				slog.Error("failed to fetch remote fixture", "url", fullURL, "error", err)
				continue
			}
			if found {
				slog.Info("using remote response fixture", "url", fullURL)
				return stripInternalMeta(payload)
			}
		}
	}

	domain := ResolveDomain(asMap(requestBody["context"]))
	return stripInternalMeta(ReadDomainResponse(domain, action, persona))
}

// ReadDomainResponse reads a legacy-tree fixture at webhook/jsons/<domain>/response/[<persona>/]<action>.json.
func ReadDomainResponse(domain string, action string, persona string) interface{} {
	normalizedDomain := NormalizeDomain(domain)
	if normalizedDomain == "" {
		slog.Warn("no response domain found, returning empty object")
		return map[string]interface{}{}
	}

	if persona != "" {
		personaPath := filepath.Join(legacyResponsesBasePath, normalizedDomain, "response", persona, fmt.Sprintf("%s.json", action))
		payload, err := readJSONFile(personaPath)
		if err == nil {
			return payload
		}
		if !os.IsNotExist(err) {
			slog.Error("failed to read fixture", "path", personaPath, "error", err)
			return map[string]interface{}{}
		}
		// fall through to default path if persona file not found
	}

	targetPath := filepath.Join(legacyResponsesBasePath, normalizedDomain, "response", fmt.Sprintf("%s.json", action))
	payload, err := readJSONFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Warn("fixture file not found, returning empty object", "path", targetPath)
			return map[string]interface{}{}
		}
		slog.Error("failed to read fixture", "path", targetPath, "error", err)
		return map[string]interface{}{}
	}
	return payload
}
