package target

func inferKey(parts []targetPart, targetPattern string) []targetPart {
	if !shouldInferKey(parts) {
		return parts
	}

	keyParts, dynamicEntry := keySourceParts(parts)
	if len(keyParts) == 0 {
		panic("protocol label target must include a resource lookup key: " + targetPattern)
	}

	key := make([]part, 0, len(keyParts))
	for _, part := range keyParts {
		key = append(key, targetPartKeyPart(part, targetPattern))
	}

	inferred := make([]targetPart, 0, inferredPartCount(dynamicEntry))
	inferred = append(inferred, parts[:resourceTargetSelectorPartCount]...)
	inferred = append(inferred, targetPart{kind: targetPartKey, key: key})
	if dynamicEntry != nil {
		inferred = append(inferred, *dynamicEntry)
	}
	return inferred
}

func shouldInferKey(parts []targetPart) bool {
	return len(parts) >= minResourceTargetPartCount &&
		parts[targetProtocolPart].kind == targetPartLiteral &&
		isProtocolPath(parts[targetProtocolPart].literal) &&
		parts[targetLookupKeyPart].kind != targetPartKey
}

func keySourceParts(parts []targetPart) ([]targetPart, *targetPart) {
	keyEnd := len(parts)
	last := parts[len(parts)-1]
	if isDynamicEntryPart(last) {
		keyEnd--
		return parts[targetLookupKeyPart:keyEnd], &last
	}
	return parts[targetLookupKeyPart:keyEnd], nil
}

func isDynamicEntryPart(part targetPart) bool {
	return part.kind == targetPartCapture && part.capture != defaultCapture
}

func inferredPartCount(dynamicEntry *targetPart) int {
	if dynamicEntry == nil {
		return compiledTargetPartCount
	}
	return compiledTargetWithDynamicEntryCount
}

func targetPartKeyPart(target targetPart, targetPattern string) part {
	switch target.kind {
	case targetPartLiteral:
		return part{kind: partLiteral, literal: target.literal}
	case targetPartCapture:
		return part{kind: partCapture, capture: target.capture}
	case targetPartDomainIndex:
		return part{kind: partDomainIndex, capture: target.capture}
	default:
		panic("nested target lookup key in label schema pattern " + targetPattern)
	}
}
