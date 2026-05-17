package target

const (
	ProtocolHTTP = "http"
	ProtocolTCP  = "tcp"
	ProtocolUDP  = "udp"

	CollectionRouters           = "routers"
	CollectionServices          = "services"
	CollectionServersTransports = "serverstransports"

	TLSDomainSegment = "domains"
	TLSDomainMain    = "main"
	TLSDomainSANs    = "sans"

	defaultCapture = "default"
)

func isProtocolPath(path string) bool {
	return path == ProtocolHTTP || path == ProtocolTCP || path == ProtocolUDP
}

func isResourceCollection(path string) bool {
	return path == CollectionRouters || path == CollectionServices || path == CollectionServersTransports
}
