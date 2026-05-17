package target

func validate(parts []targetPart, source CaptureNames, targetPattern string) {
	validateShape(parts, targetPattern)

	for _, part := range parts {
		switch part.kind {
		case targetPartCapture:
			if part.capture == defaultCapture {
				continue
			}
			if !source.HasString(part.capture) {
				panic("label target capture {" + part.capture + "} is not present in source pattern: " + targetPattern)
			}
		case targetPartKey:
			validateKeyCaptures(part.key, source, targetPattern)
		}
	}
}

func validateKeyCaptures(parts []part, source CaptureNames, targetPattern string) {
	for _, part := range parts {
		switch part.kind {
		case partCapture:
			if part.capture == defaultCapture {
				continue
			}
			if !source.HasString(part.capture) {
				panic("label target capture {" + part.capture + "} is not present in source pattern: " + targetPattern)
			}
		case partDomainIndex:
			if !source.HasInt(part.capture) {
				panic("label target domain capture {" + part.capture + "} is not present in source pattern: " + targetPattern)
			}
		}
	}
}

func validateShape(parts []targetPart, targetPattern string) {
	if len(parts) == 0 {
		panic("empty label target pattern")
	}

	first := parts[0]
	if first.kind != targetPartLiteral {
		panic("label target must start with a literal segment: " + targetPattern)
	}

	if !isProtocolPath(first.literal) {
		if len(parts) != rootTargetPartCount {
			panic("label target must be a root value or protocol resource path: " + targetPattern)
		}
		return
	}

	if len(parts) < minResourceTargetPartCount {
		panic("protocol label target must include protocol.collection.name.key: " + targetPattern)
	}
	if parts[targetCollectionPart].kind != targetPartLiteral || !isResourceCollection(parts[targetCollectionPart].literal) {
		panic("protocol label target has invalid resource collection: " + targetPattern)
	}
	if parts[targetResourceNamePart].kind != targetPartCapture {
		panic("protocol label target must use a capture as resource name: " + targetPattern)
	}
	if parts[targetLookupKeyPart].kind != targetPartKey {
		panic("protocol label target must include a resource lookup key: " + targetPattern)
	}
	if len(parts) > maxCompiledResourceTargetPartCount {
		panic("protocol label target supports only one dynamic suffix after the resource lookup key: " + targetPattern)
	}
	if len(parts) == compiledTargetWithDynamicEntryCount && !isDynamicEntryPart(parts[targetDynamicEntryPart]) {
		panic("protocol label target dynamic suffix must be a source capture: " + targetPattern)
	}
}
