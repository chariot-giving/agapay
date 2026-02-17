// Package ipfs provides a client for pinning and retrieving encrypted payment
// data on IPFS. It supports both a local IPFS node (HTTP API) and an in-memory
// mock for testing and demo purposes.
package ipfs

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
)

// Client interacts with an IPFS node to pin and retrieve data.
type Client struct {
	apiURL     string
	httpClient *http.Client
}

// NewClient creates a new IPFS client pointing to the given API URL.
// For a local IPFS node, this is typically "http://localhost:5001".
func NewClient(apiURL string) *Client {
	return &Client{
		apiURL:     apiURL,
		httpClient: &http.Client{},
	}
}

// AddResponse is the response from the IPFS add endpoint.
type AddResponse struct {
	Name string `json:"Name"`
	Hash string `json:"Hash"`
	Size string `json:"Size"`
}

// Pin uploads data to IPFS and returns the CID (Content Identifier).
func (c *Client) Pin(data []byte) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", "data")
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return "", fmt.Errorf("write data: %w", err)
	}
	writer.Close()

	url := c.apiURL + "/api/v0/add"
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("IPFS add request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("IPFS add returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var addResp AddResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		return "", fmt.Errorf("parse add response: %w", err)
	}

	return addResp.Hash, nil
}

// Retrieve fetches data from IPFS by CID.
func (c *Client) Retrieve(cid string) ([]byte, error) {
	url := c.apiURL + "/api/v0/cat?arg=" + cid
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("IPFS cat request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IPFS cat returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return data, nil
}

// PinJSON serializes a value to JSON and pins it to IPFS.
func (c *Client) PinJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal JSON: %w", err)
	}
	return c.Pin(data)
}

// RetrieveJSON fetches data from IPFS and unmarshals it from JSON.
func (c *Client) RetrieveJSON(cid string, v interface{}) error {
	data, err := c.Retrieve(cid)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// MockClient is an in-memory IPFS mock for testing and demo purposes.
// It stores data by a hash-based pseudo-CID.
type MockClient struct {
	mu    sync.RWMutex
	store map[string][]byte
}

// NewMockClient creates a new in-memory IPFS mock.
func NewMockClient() *MockClient {
	return &MockClient{
		store: make(map[string][]byte),
	}
}

// Pin stores data in memory and returns a pseudo-CID based on SHA-256.
func (m *MockClient) Pin(data []byte) (string, error) {
	hash := sha256.Sum256(data)
	cid := "bafy" + hex.EncodeToString(hash[:16])

	m.mu.Lock()
	m.store[cid] = make([]byte, len(data))
	copy(m.store[cid], data)
	m.mu.Unlock()

	return cid, nil
}

// Retrieve fetches data from the in-memory store by CID.
func (m *MockClient) Retrieve(cid string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, ok := m.store[cid]
	if !ok {
		return nil, fmt.Errorf("CID not found: %s", cid)
	}

	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// PinJSON serializes to JSON and pins to the mock store.
func (m *MockClient) PinJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal JSON: %w", err)
	}
	return m.Pin(data)
}

// RetrieveJSON fetches and unmarshals from the mock store.
func (m *MockClient) RetrieveJSON(cid string, v interface{}) error {
	data, err := m.Retrieve(cid)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// IPFSStore defines the interface for IPFS operations,
// satisfied by both Client and MockClient.
type IPFSStore interface {
	Pin(data []byte) (string, error)
	Retrieve(cid string) ([]byte, error)
	PinJSON(v interface{}) (string, error)
	RetrieveJSON(cid string, v interface{}) error
}
