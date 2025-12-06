package slipok

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type SlipOKResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Success       bool   `json:"success"`
		Message       string `json:"message"`
		TransRef      string `json:"transRef"`
		SendingBank   string `json:"sendingBank"`
		ReceivingBank string `json:"receivingBank"`
		TransDate     string `json:"transDate"`
		TransTime     string `json:"transTime"`
		Sender        struct {
			DisplayName string `json:"displayName"`
			Name        string `json:"name"`
			Proxy       struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"proxy"`
			Account struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"account"`
		} `json:"sender"`
		Receiver struct {
			DisplayName string `json:"displayName"`
			Name        string `json:"name"`
			Proxy       struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"proxy"`
			Account struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"account"`
		} `json:"receiver"`
		Amount            float64 `json:"amount"`
		PaidLocalAmount   float64 `json:"paidLocalAmount"`
		PaidLocalCurrency string  `json:"paidLocalCurrency"`
		CountryCode       string  `json:"countryCode"`
		TransFeeAmount    float64 `json:"transFeeAmount"`
		Ref1              string  `json:"ref1"`
		Ref2              string  `json:"ref2"`
		Ref3              string  `json:"ref3"`
		ToMerchantId      string  `json:"toMerchantId"`
	} `json:"data"`
}

func VerifySlip(filePath string) (*SlipOKResponse, error) {
	branchID := os.Getenv("SLIPOK_BRANCH_ID") // Optional, mostly for organization
	apiKey := os.Getenv("SLIPOK_API_KEY")

	if apiKey == "" {
		return nil, fmt.Errorf("SLIPOK_API_KEY is not set")
	}

	url := "https://api.slipok.com/api/line/apikey/" + branchID

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}
	writer.Close()

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("x-authorization", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read body for error message
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("SlipOK API returned status: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var slipResponse SlipOKResponse
	if err := json.NewDecoder(resp.Body).Decode(&slipResponse); err != nil {
		return nil, err
	}

	return &slipResponse, nil
}
