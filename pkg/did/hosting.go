package did

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// HostingService serves DID Documents for CNAME-delegated nonprofit domains.
// In the Agapay network, nonprofits delegate their DID hosting to Chariot via
// a DNS CNAME record (e.g., agapay.redcross.org CNAME dids.givechariot.com).
// This service responds to /.well-known/did.json requests for those domains.
type HostingService struct {
	mu   sync.RWMutex
	docs map[string]*Document // keyed by domain (e.g., "agapay.redcross.org")
	keys map[string]*KeyPair  // private keys, keyed by domain
}

// NewHostingService creates a new DID hosting service.
func NewHostingService() *HostingService {
	return &HostingService{
		docs: make(map[string]*Document),
		keys: make(map[string]*KeyPair),
	}
}

// RegisterDID generates a keypair and DID Document for the given domain
// and stores it for serving. Returns the generated keypair.
func (h *HostingService) RegisterDID(domain string, serviceEndpoint string) (*KeyPair, *Document, error) {
	kp, err := GenerateKeyPair()
	if err != nil {
		return nil, nil, fmt.Errorf("generate keypair for %s: %w", domain, err)
	}

	doc := CreateDIDWebDocument(domain, kp, serviceEndpoint)

	h.mu.Lock()
	h.docs[domain] = doc
	h.keys[domain] = kp
	h.mu.Unlock()

	return kp, doc, nil
}

// RegisterExistingDID stores an existing keypair and DID Document for serving.
func (h *HostingService) RegisterExistingDID(domain string, kp *KeyPair, doc *Document) {
	h.mu.Lock()
	h.docs[domain] = doc
	h.keys[domain] = kp
	h.mu.Unlock()
}

// GetDocument returns the DID Document for a given domain.
func (h *HostingService) GetDocument(domain string) (*Document, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	doc, ok := h.docs[domain]
	return doc, ok
}

// GetKeyPair returns the KeyPair for a given domain.
func (h *HostingService) GetKeyPair(domain string) (*KeyPair, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	kp, ok := h.keys[domain]
	return kp, ok
}

// ListDomains returns all registered domains.
func (h *HostingService) ListDomains() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	domains := make([]string, 0, len(h.docs))
	for d := range h.docs {
		domains = append(domains, d)
	}
	return domains
}

// Handler returns an http.Handler that serves DID Documents.
// It expects requests to arrive at /.well-known/did.json and uses the
// Host header to determine which DID Document to serve.
func (h *HostingService) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/did.json" {
			http.NotFound(w, r)
			return
		}

		// Extract the domain from the Host header.
		// The CNAME ensures the Host header matches the nonprofit's subdomain.
		host := r.Host
		if idx := strings.IndexByte(host, ':'); idx != -1 {
			host = host[:idx]
		}

		doc, ok := h.GetDocument(host)
		if !ok {
			http.Error(w, fmt.Sprintf("no DID document for domain: %s", host), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/did+ld+json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		json.NewEncoder(w).Encode(doc)
	})
}
