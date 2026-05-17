package target

import "strings"

type Context struct {
	DefaultName string
}

type Target struct {
	Key        string
	Protocol   string
	Collection string
	Name       string
	Entry      string
	Domain     *Domain
	Resource   bool
}

type Domain struct {
	Prefix string
	Index  int
	Field  string
}

func Compile(pattern string, source CaptureNames) func(Match, Context) Target {
	parts := parsePattern(pattern)
	parts = inferKey(parts, pattern)
	validate(parts, source, pattern)
	return compile(parts)
}

func compile(parts []targetPart) func(Match, Context) Target {
	return func(match Match, context Context) Target {
		if len(parts) == rootTargetPartCount &&
			parts[targetProtocolPart].kind == targetPartLiteral &&
			!isProtocolPath(parts[targetProtocolPart].literal) {
			return Target{Key: parts[targetProtocolPart].literal}
		}

		key, domain := compileKey(parts[targetLookupKeyPart].key, match, context)
		target := Target{
			Key:        key,
			Protocol:   parts[targetProtocolPart].literal,
			Collection: parts[targetCollectionPart].literal,
			Name:       resolveCapture(parts[targetResourceNamePart].capture, match, context),
			Domain:     domain,
			Resource:   true,
		}
		if len(parts) == compiledTargetWithDynamicEntryCount {
			target.Entry = resolveCapture(parts[targetDynamicEntryPart].capture, match, context)
		}
		return target
	}
}

func compileKey(parts []part, match Match, context Context) (string, *Domain) {
	segments := make([]string, 0, len(parts))
	var domain *Domain

	for index, part := range parts {
		switch part.kind {
		case partLiteral:
			segments = append(segments, part.literal)
		case partCapture:
			segments = append(segments, resolveCapture(part.capture, match, context))
		case partDomainIndex:
			segments = append(segments, TLSDomainSegment)
			if domain == nil && index+1 < len(parts) {
				field := domainField(parts[index+1])
				if field != "" && len(parts) == index+2 {
					prefix := strings.Join(segments[:len(segments)-1], ".")
					domain = &Domain{
						Prefix: prefix,
						Index:  match.Int(part.capture),
						Field:  field,
					}
				}
			}
		}
	}

	return strings.Join(segments, "."), domain
}

func domainField(part part) string {
	if part.kind != partLiteral {
		return ""
	}
	switch part.literal {
	case TLSDomainMain, TLSDomainSANs:
		return part.literal
	default:
		return ""
	}
}

func resolveCapture(capture string, match Match, context Context) string {
	if capture == defaultCapture {
		return context.DefaultName
	}
	return match.String(capture)
}
