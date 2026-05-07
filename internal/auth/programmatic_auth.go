package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	programmaticSignInEndpoint = "/iam/v1/signinProgrammaticUser" // lowercase 'i'
)

type ProgrammaticAuthConfig struct {
	BaseURL   string
	AccountID string
	AccessKey string
	SecretKey string
}

type ProgrammaticSignInRequest struct {
	AccountID string `json:"accountId"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

type ProgrammaticSignInResponse struct {
	Token   string `json:"token"`   // "token" not "access_token"
	Message string `json:"message,omitempty"`
}

func SignInProgrammatic(config ProgrammaticAuthConfig) (*ProgrammaticSignInResponse, error) {
	reqBody := ProgrammaticSignInRequest{
		AccountID: config.AccountID,
		AccessKey: config.AccessKey,
		SecretKey: config.SecretKey,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		config.BaseURL+programmaticSignInEndpoint,
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("signin failed (%s): %s", resp.Status, string(b))
	}

	var signInResp ProgrammaticSignInResponse
	if err := json.NewDecoder(resp.Body).Decode(&signInResp); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	if signInResp.Token == "" {
		return nil, fmt.Errorf("token is empty in signin response")
	}

	return &signInResp, nil
}