package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)



const (
	signInEndpoint     = "/iam/v1/signInAsIAMUser"
	rootSignInEndpoint = "/iam/v1/signInAsRootUser"
)



type AuthConfig struct {
	BaseURL    string
	Email      string
	Password   string
	AccountID  string
	IsRootUser bool
}



type SignInRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	AccountID string `json:"accountId,omitempty"`
}

type SignInResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Message      string `json:"message,omitempty"`
	MFAEnabled   int    `json:"mfaEnabled,omitempty"`
}




func SignIn(config AuthConfig) (*SignInResponse, error) {
	var endpoint string


	reqBody := SignInRequest{
		Email:    config.Email,
		Password: config.Password,
	}

	if config.IsRootUser {
		endpoint = rootSignInEndpoint
	} else {
		endpoint = signInEndpoint
		reqBody.AccountID = config.AccountID
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sign-in request: %w", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		config.BaseURL+endpoint,
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create sign-in request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send sign-in request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"sign-in failed (%s): %s",
			resp.Status,
			string(b),
		)
	}

	var signInResp SignInResponse
	if err := json.NewDecoder(resp.Body).Decode(&signInResp); err != nil {
		return nil, fmt.Errorf("failed to decode sign-in response: %w", err)
	}

	if signInResp.AccessToken == "" {
		return nil, fmt.Errorf("sign-in succeeded but access token is empty")
	}

	return &signInResp, nil
}
