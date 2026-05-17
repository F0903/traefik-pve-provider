package target

// Compiled targets use a fixed shape:
// protocol.collection.{name}.key[.{entry}]
const (
	targetProtocolPart = iota
	targetCollectionPart
	targetResourceNamePart
	targetLookupKeyPart
	targetDynamicEntryPart
)

const (
	rootTargetPartCount                 = 1
	resourceTargetSelectorPartCount     = targetLookupKeyPart
	minResourceTargetPartCount          = targetLookupKeyPart + 1
	maxCompiledResourceTargetPartCount  = targetDynamicEntryPart + 1
	compiledTargetPartCount             = targetLookupKeyPart + 1
	compiledTargetWithDynamicEntryCount = targetDynamicEntryPart + 1
)
