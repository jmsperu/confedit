package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// FileType represents the detected config file format.
type FileType string

const (
	TypeNginx    FileType = "nginx"
	TypeApache   FileType = "apache"
	TypeSSH      FileType = "ssh"
	TypeINI      FileType = "ini"
	TypeEnv      FileType = "env"
	TypeYAML     FileType = "yaml"
	TypeJSON     FileType = "json"
	TypeTOML     FileType = "toml"
	TypeSysctl   FileType = "sysctl"
	TypeHosts    FileType = "hosts"
	TypeUnknown  FileType = "unknown"
)

// KeyValue represents a config key-value pair.
type KeyValue struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Line    int    `json:"line"`
	Section string `json:"section,omitempty"`
	Comment string `json:"comment,omitempty"`
}

// DetectType detects the config file type from filename and content.
func DetectType(filename string) FileType {
	lower := strings.ToLower(filename)

	if strings.Contains(lower, "nginx") || strings.HasSuffix(lower, ".conf") && strings.Contains(lower, "nginx") {
		return TypeNginx
	}
	if strings.Contains(lower, "apache") || strings.Contains(lower, "httpd") || strings.HasSuffix(lower, ".htaccess") {
		return TypeApache
	}
	if strings.Contains(lower, "ssh") || lower == "sshd_config" || lower == "ssh_config" || strings.HasSuffix(lower, "/config") {
		return TypeSSH
	}
	if strings.HasSuffix(lower, ".ini") || strings.HasSuffix(lower, ".cfg") {
		return TypeINI
	}
	if strings.HasSuffix(lower, ".env") || strings.Contains(lower, ".env.") {
		return TypeEnv
	}
	if strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml") {
		return TypeYAML
	}
	if strings.HasSuffix(lower, ".json") {
		return TypeJSON
	}
	if strings.HasSuffix(lower, ".toml") {
		return TypeTOML
	}
	if strings.Contains(lower, "sysctl") {
		return TypeSysctl
	}
	if lower == "hosts" || strings.HasSuffix(lower, "/hosts") {
		return TypeHosts
	}

	return TypeUnknown
}

// ParseFile reads a config file and extracts key-value pairs.
func ParseFile(filename string) ([]KeyValue, FileType, error) {
	fileType := DetectType(filename)

	switch fileType {
	case TypeYAML:
		return parseYAML(filename)
	case TypeJSON:
		return parseJSON(filename)
	case TypeEnv:
		return parseEnv(filename)
	case TypeINI, TypeSysctl:
		return parseINI(filename)
	case TypeSSH:
		return parseSSH(filename)
	case TypeHosts:
		return parseHosts(filename)
	default:
		return parseGeneric(filename)
	}
}

func parseEnv(filename string) ([]KeyValue, FileType, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, TypeEnv, err
	}
	defer f.Close()

	var kvs []KeyValue
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.IndexByte(line, '='); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			// Remove quotes
			value = strings.Trim(value, "\"'")
			kvs = append(kvs, KeyValue{Key: key, Value: value, Line: lineNum})
		}
	}

	return kvs, TypeEnv, nil
}

func parseINI(filename string) ([]KeyValue, FileType, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, TypeINI, err
	}
	defer f.Close()

	var kvs []KeyValue
	scanner := bufio.NewScanner(f)
	lineNum := 0
	section := ""

	sectionRe := regexp.MustCompile(`^\[(.+)\]$`)

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if m := sectionRe.FindStringSubmatch(line); m != nil {
			section = m[1]
			continue
		}

		if idx := strings.IndexAny(line, "=:"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			kvs = append(kvs, KeyValue{Key: key, Value: value, Line: lineNum, Section: section})
		}
	}

	return kvs, TypeINI, nil
}

func parseSSH(filename string) ([]KeyValue, FileType, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, TypeSSH, err
	}
	defer f.Close()

	var kvs []KeyValue
	scanner := bufio.NewScanner(f)
	lineNum := 0
	section := ""

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "Host ") || strings.HasPrefix(line, "Match ") {
			section = line
			kvs = append(kvs, KeyValue{Key: strings.Fields(line)[0], Value: strings.Join(strings.Fields(line)[1:], " "), Line: lineNum, Section: section})
			continue
		}

		fields := strings.SplitN(line, " ", 2)
		if len(fields) == 2 {
			kvs = append(kvs, KeyValue{Key: strings.TrimSpace(fields[0]), Value: strings.TrimSpace(fields[1]), Line: lineNum, Section: section})
		} else if idx := strings.IndexByte(line, '='); idx > 0 {
			kvs = append(kvs, KeyValue{Key: strings.TrimSpace(line[:idx]), Value: strings.TrimSpace(line[idx+1:]), Line: lineNum, Section: section})
		}
	}

	return kvs, TypeSSH, nil
}

func parseHosts(filename string) ([]KeyValue, FileType, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, TypeHosts, err
	}
	defer f.Close()

	var kvs []KeyValue
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			ip := fields[0]
			for _, host := range fields[1:] {
				if strings.HasPrefix(host, "#") {
					break
				}
				kvs = append(kvs, KeyValue{Key: host, Value: ip, Line: lineNum})
			}
		}
	}

	return kvs, TypeHosts, nil
}

func parseYAML(filename string) ([]KeyValue, FileType, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, TypeYAML, err
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, TypeYAML, fmt.Errorf("invalid YAML: %w", err)
	}

	var kvs []KeyValue
	flattenMap("", raw, &kvs)
	return kvs, TypeYAML, nil
}

func parseJSON(filename string) ([]KeyValue, FileType, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, TypeJSON, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, TypeJSON, fmt.Errorf("invalid JSON: %w", err)
	}

	var kvs []KeyValue
	flattenMap("", raw, &kvs)
	return kvs, TypeJSON, nil
}

func flattenMap(prefix string, m map[string]interface{}, kvs *[]KeyValue) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		switch val := v.(type) {
		case map[string]interface{}:
			flattenMap(key, val, kvs)
		case []interface{}:
			for i, item := range val {
				itemKey := fmt.Sprintf("%s[%d]", key, i)
				if subMap, ok := item.(map[string]interface{}); ok {
					flattenMap(itemKey, subMap, kvs)
				} else {
					*kvs = append(*kvs, KeyValue{Key: itemKey, Value: fmt.Sprintf("%v", item)})
				}
			}
		default:
			*kvs = append(*kvs, KeyValue{Key: key, Value: fmt.Sprintf("%v", val)})
		}
	}
}

func parseGeneric(filename string) ([]KeyValue, FileType, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, TypeUnknown, err
	}
	defer f.Close()

	var kvs []KeyValue
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") {
			continue
		}

		// Try key=value
		if idx := strings.IndexByte(line, '='); idx > 0 {
			kvs = append(kvs, KeyValue{
				Key:   strings.TrimSpace(line[:idx]),
				Value: strings.TrimSpace(line[idx+1:]),
				Line:  lineNum,
			})
			continue
		}

		// Try key value (space separated like nginx)
		fields := strings.Fields(line)
		if len(fields) >= 2 && !strings.HasSuffix(fields[0], "{") {
			key := fields[0]
			value := strings.Join(fields[1:], " ")
			value = strings.TrimSuffix(value, ";")
			kvs = append(kvs, KeyValue{Key: key, Value: value, Line: lineNum})
		}
	}

	return kvs, TypeUnknown, nil
}

// SetValue modifies a value in a config file.
func SetValue(filename, key, newValue string) error {
	fileType := DetectType(filename)

	switch fileType {
	case TypeYAML:
		return setYAMLValue(filename, key, newValue)
	case TypeJSON:
		return setJSONValue(filename, key, newValue)
	case TypeEnv:
		return setLineValue(filename, key, newValue, "=", "")
	case TypeINI, TypeSysctl:
		return setLineValue(filename, key, newValue, "=", "")
	case TypeSSH:
		return setLineValue(filename, key, newValue, " ", "")
	default:
		return setLineValue(filename, key, newValue, "=", "")
	}
}

func setLineValue(filename, key, newValue, separator, quoteChar string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	found := false
	keyLower := strings.ToLower(key)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			continue
		}

		var lineKey string
		if separator == " " {
			fields := strings.Fields(trimmed)
			if len(fields) >= 1 {
				lineKey = fields[0]
			}
		} else {
			if idx := strings.Index(trimmed, separator); idx > 0 {
				lineKey = strings.TrimSpace(trimmed[:idx])
			}
		}

		if strings.ToLower(lineKey) == keyLower {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			if quoteChar != "" {
				newValue = quoteChar + newValue + quoteChar
			}
			lines[i] = indent + key + separator + newValue
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("key %q not found in %s", key, filename)
	}

	return os.WriteFile(filename, []byte(strings.Join(lines, "\n")), 0644)
}

func setYAMLValue(filename, key, newValue string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}

	parts := strings.Split(key, ".")
	setNestedValue(raw, parts, newValue)

	out, err := yaml.Marshal(raw)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, out, 0644)
}

func setJSONValue(filename, key, newValue string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	parts := strings.Split(key, ".")
	setNestedValue(raw, parts, newValue)

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, append(out, '\n'), 0644)
}

func setNestedValue(m map[string]interface{}, keys []string, value string) {
	if len(keys) == 1 {
		m[keys[0]] = value
		return
	}
	if sub, ok := m[keys[0]].(map[string]interface{}); ok {
		setNestedValue(sub, keys[1:], value)
	}
}

// Validate performs basic validation on a config file.
func Validate(filename string) []string {
	fileType := DetectType(filename)
	var errors []string

	switch fileType {
	case TypeJSON:
		data, err := os.ReadFile(filename)
		if err != nil {
			return []string{err.Error()}
		}
		var raw interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			errors = append(errors, fmt.Sprintf("Invalid JSON: %v", err))
		}

	case TypeYAML:
		data, err := os.ReadFile(filename)
		if err != nil {
			return []string{err.Error()}
		}
		var raw interface{}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			errors = append(errors, fmt.Sprintf("Invalid YAML: %v", err))
		}

	case TypeEnv:
		f, err := os.Open(filename)
		if err != nil {
			return []string{err.Error()}
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if !strings.Contains(line, "=") {
				errors = append(errors, fmt.Sprintf("Line %d: missing '=' separator: %s", lineNum, line))
			}
		}
	}

	if len(errors) == 0 {
		errors = append(errors, "OK — no errors found")
	}
	return errors
}
