package toml

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Node any

type ParseError struct {
	Line int
	Msg  string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Msg)
}

type Parser struct {
	lines    []string
	lineNum  int
	result   map[string]any
	current  map[string]any
	tableKey []string
}

func New() *Parser {
	return &Parser{
		result:  make(map[string]any),
		current: make(map[string]any),
	}
}

// Parse converts TOML string to JSON
func Parse(tomlData string) ([]byte, error) {
	parser := New()
	if err := parser.parse(tomlData); err != nil {
		return nil, err
	}
	return json.Marshal(parser.result)
}

// ParseNative converts TOML string to native Go data structures
func ParseNative(tomlData string) (map[string]any, error) {
	parser := New()
	if err := parser.parse(tomlData); err != nil {
		return nil, err
	}
	return parser.result, nil
}

// parse main parsing logic
func (p *Parser) parse(data string) error {
	p.lines = strings.Split(data, "\n")
	p.result = make(map[string]any)
	p.current = p.result

	for p.lineNum = 0; p.lineNum < len(p.lines); p.lineNum++ {
		line := strings.TrimSpace(p.lines[p.lineNum])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if err := p.parseTable(line); err != nil {
				return err
			}
			continue
		}

		if strings.Contains(line, "=") {
			if err := p.parseKeyValue(line); err != nil {
				return err
			}
			continue
		}

		return &ParseError{Line: p.lineNum + 1, Msg: "invalid syntax"}
	}

	return nil
}

// parseTable handles table parsing [table] or [table.subtable]
func (p *Parser) parseTable(line string) error {
	key := strings.TrimSpace(line[1 : len(line)-1])
	if key == "" {
		return &ParseError{Line: p.lineNum + 1, Msg: "empty table name"}
	}

	p.tableKey = strings.Split(key, ".")
	p.current = p.result
	for _, k := range p.tableKey {
		if next, exists := p.current[k]; !exists {
			newTable := make(map[string]any)
			p.current[k] = newTable
			p.current = newTable
		} else {
			if nextMap, ok := next.(map[string]any); ok {
				p.current = nextMap
			} else {
				return &ParseError{Line: p.lineNum + 1, Msg: "key already exists"}
			}
		}
	}
	return nil
}

// parseKeyValue handles key = value parsing
func (p *Parser) parseKeyValue(line string) error {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return &ParseError{Line: p.lineNum + 1, Msg: "invalid key-value pair"}
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	parsedValue, err := p.parseValue(value)
	if err != nil {
		return &ParseError{Line: p.lineNum + 1, Msg: err.Error()}
	}

	p.current[key] = parsedValue
	return nil
}

// parseValue handles value parsing
func (p *Parser) parseValue(value string) (any, error) {
	// Handle strings
	if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return value[1 : len(value)-1], nil
	}

	// Handle numbers
	if num, err := strconv.Atoi(value); err == nil {
		return num, nil
	}
	if num, err := strconv.ParseFloat(value, 64); err == nil {
		return num, nil
	}

	// Handle booleans
	if value == "true" {
		return true, nil
	}
	if value == "false" {
		return false, nil
	}

	// Handle arrays
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		return p.parseArray(value)
	}

	// Handle datetime (basic format)
	if strings.Contains(value, "T") && len(value) >= 19 {
		if t, err := time.Parse(time.RFC3339, value); err == nil {
			return t, nil
		}
	}

	return nil, fmt.Errorf("unrecognized value: %s", value)
}

// parseArray handles array parsing
func (p *Parser) parseArray(value string) ([]any, error) {
	content := strings.TrimSpace(value[1 : len(value)-1])
	if content == "" {
		return []any{}, nil
	}

	var result []any
	var current strings.Builder
	inString := false
	escape := false

	for i, r := range content {
		// Handle string literals
		if r == '"' && !escape {
			inString = !inString
		}

		// Handle escape sequences
		if r == '\\' && !escape && inString {
			escape = true
			continue
		}

		if escape {
			escape = false
			current.WriteRune(r)
			continue
		}

		// Handle array separators
		if r == ',' && !inString {
			val := strings.TrimSpace(current.String())
			if val == "" {
				return nil, fmt.Errorf("empty array element at position %d", i)
			}
			parsed, err := p.parseValue(val)
			if err != nil {
				return nil, err
			}
			result = append(result, parsed)
			current.Reset()
			continue
		}

		current.WriteRune(r)
	}

	// Handle last element
	if current.Len() > 0 || strings.HasSuffix(content, ",") {
		val := strings.TrimSpace(current.String())
		if val == "" && !strings.HasSuffix(content, ",") {
			return nil, fmt.Errorf("empty array element")
		}
		if val != "" {
			parsed, err := p.parseValue(val)
			if err != nil {
				return nil, err
			}
			result = append(result, parsed)
		}
	}

	return result, nil
}
