package subscription

import (
	"bufio"
	"io"
	"sort"
	"strings"
)

const maxLineLength = 512
const maxDomainLength = 253

func ParseList(r io.Reader) ([]string, error) {
	seen := make(map[string]struct{})
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, maxLineLength), maxLineLength)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Skip comments
		if line[0] == '#' || line[0] == '!' {
			continue
		}

		domain := parseLine(line)
		if domain == "" {
			continue
		}

		domain = strings.ToLower(domain)
		if !isValidDomain(domain) {
			continue
		}

		seen[domain] = struct{}{}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	domains := make([]string, 0, len(seen))
	for d := range seen {
		domains = append(domains, d)
	}
	sort.Strings(domains)
	return domains, nil
}

func parseLine(line string) string {
	// AdGuard basic: ||domain.com^ or ||domain.com^$modifiers
	if strings.HasPrefix(line, "||") {
		rest := line[2:]
		if idx := strings.IndexByte(rest, '^'); idx > 0 {
			return rest[:idx]
		}
	}

	// Hosts file: 0.0.0.0 domain or 127.0.0.1 domain
	if strings.HasPrefix(line, "0.0.0.0 ") || strings.HasPrefix(line, "127.0.0.1 ") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			domain := fields[1]
			if domain == "localhost" || domain == "localhost.localdomain" {
				return ""
			}
			return domain
		}
		return ""
	}

	// Plain text: single domain per line (no spaces)
	if !strings.ContainsAny(line, " \t") {
		return line
	}

	return ""
}

func isValidDomain(domain string) bool {
	if len(domain) == 0 || len(domain) > maxDomainLength {
		return false
	}
	if strings.ContainsAny(domain, " \t/\\@:|^$*") {
		return false
	}
	if !strings.Contains(domain, ".") {
		return false
	}
	return true
}
