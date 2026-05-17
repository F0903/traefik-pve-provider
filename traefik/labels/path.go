package labels

import schematarget "github.com/F0903/traefik-pve-provider/traefik/labels/schema/target"

const (
	protocolHTTP = schematarget.ProtocolHTTP
	protocolTCP  = schematarget.ProtocolTCP
	protocolUDP  = schematarget.ProtocolUDP

	collectionRouters           = schematarget.CollectionRouters
	collectionServices          = schematarget.CollectionServices
	collectionServersTransports = schematarget.CollectionServersTransports

	rootEnable = "enable"
	rootName   = "name"

	tlsDomainSegment = schematarget.TLSDomainSegment
	tlsDomainMain    = schematarget.TLSDomainMain
	tlsDomainSANs    = schematarget.TLSDomainSANs
)
