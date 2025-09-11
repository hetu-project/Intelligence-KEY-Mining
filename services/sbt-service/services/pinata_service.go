package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/services/sbt-service/models"
)

// PinataService handles IPFS operations via Pinata
type PinataService struct {
	apiKey    string
	secretKey string
	baseURL   string
	client    *http.Client
}

// NewPinataService creates a new Pinata service
func NewPinataService(apiKey, secretKey string) *PinataService {
	return &PinataService{
		apiKey:    apiKey,
		secretKey: secretKey,
		baseURL:   "https://api.pinata.cloud",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// UploadJSON uploads JSON metadata to IPFS via Pinata
func (ps *PinataService) UploadJSON(ctx context.Context, metadata interface{}, name string) (*models.PinataResponse, error) {
	// Build request
	uploadReq := &models.PinataUploadRequest{
		PinataContent: metadata,
		PinataMetadata: &models.PinataMetadata{
			Name: name,
			KeyValues: map[string]string{
				"type":      "sbt_metadata",
				"timestamp": fmt.Sprintf("%d", time.Now().Unix()),
			},
		},
		PinataOptions: &models.PinataOptions{
			CidVersion: 1,
		},
	}

	reqBody, err := json.Marshal(uploadReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", ps.baseURL+"/pinning/pinJSONToIPFS", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("pinata_api_key", ps.apiKey)
	req.Header.Set("pinata_secret_api_key", ps.secretKey)

	// Send request
	resp, err := ps.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pinata API error: %d - %s", resp.StatusCode, string(body))
	}

	// Parse response
	var pinataResp models.PinataResponse
	if err := json.NewDecoder(resp.Body).Decode(&pinataResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &pinataResp, nil
}

// UploadImage uploads image to IPFS via Pinata
func (ps *PinataService) UploadImage(ctx context.Context, imageData []byte, filename string) (*models.PinataResponse, error) {
	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %v", err)
	}

	if _, err := part.Write(imageData); err != nil {
		return nil, fmt.Errorf("failed to write image data: %v", err)
	}

	// Add metadata
	pinataMetadata := &models.PinataMetadata{
		Name: filename,
		KeyValues: map[string]string{
			"type":      "sbt_avatar",
			"timestamp": fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	metadataJSON, _ := json.Marshal(pinataMetadata)
	writer.WriteField("pinataMetadata", string(metadataJSON))

	pinataOptions := &models.PinataOptions{
		CidVersion: 1,
	}
	optionsJSON, _ := json.Marshal(pinataOptions)
	writer.WriteField("pinataOptions", string(optionsJSON))

	writer.Close()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", ps.baseURL+"/pinning/pinFileToIPFS", &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("pinata_api_key", ps.apiKey)
	req.Header.Set("pinata_secret_api_key", ps.secretKey)

	// Send request
	resp, err := ps.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pinata API error: %d - %s", resp.StatusCode, string(body))
	}

	// Parse response
	var pinataResp models.PinataResponse
	if err := json.NewDecoder(resp.Body).Decode(&pinataResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &pinataResp, nil
}

// UploadBase64Image uploads base64 encoded image to IPFS
func (ps *PinataService) UploadBase64Image(ctx context.Context, base64Data, filename string) (*models.PinataResponse, error) {
	// Decode base64 data
	// Handle data:image/png;base64, prefix
	if strings.Contains(base64Data, ",") {
		parts := strings.Split(base64Data, ",")
		if len(parts) == 2 {
			base64Data = parts[1]
		}
	}

	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 image: %v", err)
	}

	return ps.UploadImage(ctx, imageData, filename)
}

// UnpinContent unpins content from IPFS (cleanup)
func (ps *PinataService) UnpinContent(ctx context.Context, ipfsHash string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", ps.baseURL+"/pinning/unpin/"+ipfsHash, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("pinata_api_key", ps.apiKey)
	req.Header.Set("pinata_secret_api_key", ps.secretKey)

	resp, err := ps.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pinata API error: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// TestConnection tests connection to Pinata API
func (ps *PinataService) TestConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", ps.baseURL+"/data/testAuthentication", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("pinata_api_key", ps.apiKey)
	req.Header.Set("pinata_secret_api_key", ps.secretKey)

	resp, err := ps.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pinata authentication failed: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if result.Message != "Congratulations! You are communicating with the Pinata API!" {
		return fmt.Errorf("unexpected response: %s", result.Message)
	}

	return nil
}

// GetPinnedContent gets information about pinned content
func (ps *PinataService) GetPinnedContent(ctx context.Context, ipfsHash string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/data/pinList?hashContains=%s", ps.baseURL, ipfsHash)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("pinata_api_key", ps.apiKey)
	req.Header.Set("pinata_secret_api_key", ps.secretKey)

	resp, err := ps.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pinata API error: %d - %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return result, nil
}

// Helper functions

// FormatPinataGatewayURL formats IPFS hash as Pinata gateway URL
func FormatPinataGatewayURL(ipfsHash string) string {
	return fmt.Sprintf("https://gateway.pinata.cloud/ipfs/%s", ipfsHash)
}
