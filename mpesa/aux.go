package mpesa

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"
)

const (
	//4130455:172f9892373eafe6dac71a87e4e8ade1792599809f7de1667c647bce03364ca7
	// 4904594
	ConsumerKey       = "a53D2lxIgGTgXtTDEnMo5btDnG90nOhg16GOK0MAlOOQNhBe"
	ConsumerSecret    = "LchqRZnB48pQfGB1WUvNhp6qqzGQ3MfBFd32sGsqvYIzvmJswghXXWA0KormP3NV"
	BusinessShortCode = "4130455"
	PassKey           = "172f9892373eafe6dac71a87e4e8ade1792599809f7de1667c647bce03364ca7"
	Password          = "Xw8NWgC6K4Hnese1stlIMC0sE3p+kbcMtTVVxG57s4K/WZB2owiOf30B3yYSdTaTqdz2gv22we9sd4bgvfPVl7jynLtAglZn6KuGtdhhdy3eVQ0nosw3wZdfHDum8DCu5BAI/jU+x32PMSB/vtx9bbreV0rUHEvx7Gx4CI4Eze4BnhFQ368Z2x7x9Q+82r/tZxDlgG76NbWnLfj9DHbcs5hOBoMYiMbnXg8HsLUaI688qNGqqK9CLr8uKfIgXgFBSD4Ky7P9UwWBXlTOODtmv/TRJBnrD+8IFttZqjruDxV81NGIeASl9q6Ni8go5gBGrNHGxSJ/SF5rGhloTXLtHg=="
	InitiatorName     = "b2cInit"

	// test b2c
	AppconumerKey         = "Lj4StGZWiCQRbgSQmGZV9FMEd0ZRzdkJ7h8yAhgiEdpKWFaO"
	AppconumerSecret      = "FYRQ59r2imH0Kw89SQH9qZKKJOupFTNWwEd0bt5zazqZUl8ie75bAwSSLuAtNxRh"
	AppinitiatorName      = "collins"
	AppsecurityCredential = "W0LXtRIf33TqQuSQLevyqTyvO847tqYMB3WauCFdYGRF6PiXZj770OhmG3lJnX4cVMrsm3K258oUx7y2p6MCs1V+W8YXNM3oqfb9pOoXFqnSmyTammvSetJct3/w0UT+0FUJTrg8JXH6j0FYlsqibCXc8f9ATb+twMi4Mxm37Ehu7fNOP40c6BHO7Cp4HHUa5yHjAVOSNDioWZr39bzyvBIiCT9Az/aISj060rCMZLWGlUlbpKVVQFqAwh/flu8AXxVqT2/zhg5NOPjhHb5i8uyau1IhN9LxC9nOGBKqInmlj59g4qTUPlSY1apZ3Y+Ny6iPN3yyseq3mm4UXKxGAg=="
	APPshortCode          = "3008826"

	// bc2 details
	B2Cconsumerkey       = "oLwt5LEkO7zkQaqV8Sy9Gs8MvgA8PFADM6VOUe4jYj98nVr1"
	B2Cconsumersecret    = "YylBuouNZdeOJeU8ltCKll5QBQ0xSDrdAq7pdaurpOS8FNYPkaSAA8kZLlblwslM"
	B2CBusinessShortCode = "3039805"
	B2CPassword          = "Xw8NWgC6K4Hnese1stlIMC0sE3p+kbcMtTVVxG57s4K/WZB2owiOf30B3yYSdTaTqdz2gv22we9sd4bgvfPVl7jynLtAglZn6KuGtdhhdy3eVQ0nosw3wZdfHDum8DCu5BAI/jU+x32PMSB/vtx9bbreV0rUHEvx7Gx4CI4Eze4BnhFQ368Z2x7x9Q+82r/tZxDlgG76NbWnLfj9DHbcs5hOBoMYiMbnXg8HsLUaI688qNGqqK9CLr8uKfIgXgFBSD4Ky7P9UwWBXlTOODtmv/TRJBnrD+8IFttZqjruDxV81NGIeASl9q6Ni8go5gBGrNHGxSJ/SF5rGhloTXLtHg=="
)

// vuka creds
const (
	//4130455:172f9892373eafe6dac71a87e4e8ade1792599809f7de1667c647bce03364ca7
	// 4904594
	VukaC2BConsumerKey       = "JYYWwClNVsMWO3IGCjvvN9TnpvvmSNI0BrldPQr81lnHWVHj"
	VukaC2BConsumerSecret    = "FHRt2bBkIlgCTsrAAcWKH6mIe9faO283YMrytFnzKjJrTqUArlMJsBWHWEifg83w"
	VukaC2BBusinessShortCode = "4041587"
	VukaC2BPassKey           = "1f441ccbc8e477a4e24094d603f172fc08620b3fa104ea14e206aa0465ad7d07"

	VukaPayB2CConsumerKey    = "FYAzZv4GvPsYpIG0Yxan3k9llRAcv59HAnwP62pbr6gabOqf"
	VukaPayB2CConsumerSecret = "3IrR0Q0qbRnkhl2L2PB3oWDujrZpMvg00F7hYFBoihZGMpXuObCuKPzlFPIkJM2V"
	VukaPayB2CInitiatorName  = "Collin"
	VukaPayB2CPassword       = "jUdSHSh84lzrYUnmIwfiZZIrOL7+o0sRRxteBLEJLO60lHVfV7K10ySoE0E8EqvbU6u6ZMNh6ATfQf8sU+XbFnWdMZUlADuhJXeUeGMk8Z842l8J8kWC3txYM1U0X5qDf3K/QnU26kj4UiRqhkXaIjJ69SL26ptVFozFYI2+8WXOH6Hhj20dDhWfsNaJCl8gYeAqdJMockmsZ1PQYNe6oph2jFPTS5kRKuXOglIYtVe97xkIdsnzKScseqTFRxm6Anlroi0fZLP9svNbOANSqTWY0p5rtuyILZlUD/gzWbAVlvO5SImLqI0RIikzAAuxnXvGkaKw36V795ItSwdeRQ=="
	VukaPayB2CShortCode      = "3008816"
)

const (

	// 4904594
	AppC2BConsumerKey       = "AKCwOp24DxNCotKUIjZzPjGgVXqJ4izSa5jyF9JDTP6XHGSC"
	AppC2BConsumerSecret    = "Gg3c5rzpNpKGJR0ZJ64U0JoGCfo4VO0cytBS0HnljAZEoctYS4a7EAGUcgLxVG8W"
	AppC2BBusinessShortCode = "4041529"
	AppC2BPassKey           = "bd160634242c28b805468346e4647c22ae7cd9b2dc46e88ec193d89ce8a16146"

	AppPayB2CConsumerKey    = "m99dbV8i4Vm4GgIn9yQ903a5kOZoi9DXClmFkVq4Aepo3ihx"
	AppPayB2CConsumerSecret = "x9CJvif8SpRg96qX0cUSyfyCrEkWjIjjwtoH5kGRIUE38orM714VImejkMDPs1EN"
	AppPayB2CInitiatorName  = "Anuar"
	AppPayB2CPassword       = "ZaZL3V2T1zRirqevhpYDSRBGvstw7kg+V/W6w9v6ipFmIIlYEubYb0lcjFPsS47GS8KzxfBeFs3rtgoKF/ZgIGSnsTXSNVwPR+1r50ypxXjwl722doeUolmkMRaeonsHcVACASz8CwGXg/STXoU14MxB4WejmM//5hc+9VSaqCoSBGTmNxQbethuQOk5JsQ9YNi1JSDb5eFOzAO6lj4/BBxIhsOFPdAF3maxZHPdxAvjGK9DLEYvOGFQLv9ll/j5IzwnoM0/KVH8k2fJFgXlTfmGw2S72GxHmd/HcXKARKSyfDQos9ZFoTrvlZH0Vt8qHpZYbqPviuzyHjDEm8IwNw=="
	AppPayB2CShortCode      = "3008818"
)

// technology@crayfinance.com
const (
	//4130455:172f9892373eafe6dac71a87e4e8ade1792599809f7de1667c647bce03364ca7
	// 4904594
	CrayC2BConsumerKey       = "v1PsGtti6d1GaS1JYV9Txo2S5dEI3Ea3V6SdCQ9B98HVuXW5"
	CrayC2BConsumerSecret    = "3JP0v5g0AJm2GEUAVDOD9sbxp1XPpQwNhAoAqZFBbBTt0JMXDHGIs46X7NkrVQAo"
	CrayC2BBusinessShortCode = "4041603"
	CrayC2BPassKey           = "4f2269d5d4270a073d41f9f9b72260dfa5265c78eae65cf5f2635bc06883e0fe"

	CrayPayB2CConsumerKey    = "3RNMVF7lei58Sm3xGGJv4qkTgz3laFZ3zXi7BI7JjE5pasq5"
	CrayPayB2CConsumerSecret = "pe5nTUfjgmMXnA8AQ1OX7vuILL7nPZOGqG9JFrTQPOYtDAuQrQBu9kmOcx0TdcLJ"
	CrayPayB2CInitiatorName  = "collins"
	CrayPayB2CPassword       = "P8tMtdc3GunIzpxqKoF4FC8SbPIGvV5lMtoqWMZvdQ5aKeRzWqQItVqAs8BDXSHoJS+kUZpjJ7xAH4q0W/L3Wv421RiI7VpTHk7TfcSFVPnwL1dYYUVPaSSuZPSMAgedkGIfL/kR8Uc9Q+pE4CMS7bCkDUdHfm61YiNgFQ+mabs0aFPynI4hYXEYGAXc4VIMQ+hevhKDs4cKvddrVGoGVYapi+vjqdzMe25gnyp6pZdHT+y6Lkm09WhIkmgRsVOSZnXixmPLEIgTjdw+xLHGFIqYn+QDE0Mk2jahNVXi82P2BGZ4ouORSHPrwE7nLOCKyiZUcfZKbok44uO+9BTjgg=="
	CrayPayB2CShortCode      = "3008814"
)

// TWD
const (
	TWDC2BConsumerKey       = "P2BwQcGj8fnnvsgirs2iFZ6pWbjLsrAJEzIlO2vvOnHM4uHS"
	TWDC2BConsumerSecret    = "26RPkoUxX4RRa9Wywg5RAQFOGwZwZYxjTzCIlO3kj6it56J9ze40TUxLvYMGj4Gd"
	TWDC2BBusinessShortCode = "4041809"
	TWDC2BPassKey           = "ef99e90e1744e1a689df2a3c2bcee521caf8c93d7db8f723c7634ad8424db324"

	TWDPayB2CConsumerKey    = "FYAzZv4GvPsYpIG0Yxan3k9llRAcv59HAnwP62pbr6gabOqf"
	TWDPayB2CConsumerSecret = "3IrR0Q0qbRnkhl2L2PB3oWDujrZpMvg00F7hYFBoihZGMpXuObCuKPzlFPIkJM2V"
	TWDPayB2CInitiatorName  = "collins"
	TWDPayB2CPassword       = "AB2K1KfElkp1OY9VRAixm3qE99wo//IuJWs0CVdUMdq5L5zPjeGcWrlIdhZdErO9SVGQ5Ba3kVvmPw9Cw1xtH1zKBVd9sY4byCQj9Hbr5qfnjk3wjV8WGw/1gtGI7SA+aRPdelKSnwlbO+6JMX2fHr+0YcBWkaRV8JxoQBuKWbk0jvy+mh8u822M69kiUCphzbT4uO1p6mXm++ahK2oVMc1UBJC6OgxMHep2zDvm45hqsnTgJYHds8xrgFbXomg6MEiqEfuf4XXexogtGv3YXewYXsyy2uhFshLj+aRipANstEsLGzC4+nd5Grr5aLFQQhJ4iviZEGpD+mHDS78sZA=="
	TWDPayB2CShortCode      = "3008816"
)

// revert amout  using the api

type StkPushResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
}

type B2BResponse struct {
	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

// Structs for the JSON payload and responses
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

type StkPushRequest struct {
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	TransactionType   string `json:"TransactionType"`
	Amount            int    `json:"Amount"`
	PartyA            string `json:"PartyA"`
	PartyB            string `json:"PartyB"`
	PhoneNumber       string `json:"PhoneNumber"`
	CallBackURL       string `json:"CallBackURL"`
	AccountReference  string `json:"AccountReference"`
	TransactionDesc   string `json:"TransactionDesc"`
}

func GenerateAccessToken(consumerKey, consumerSecret string) (string, error) {
	url := "https://api.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials"

	// Create the basic auth header
	credentials := base64.StdEncoding.EncodeToString([]byte(consumerKey + ":" + consumerSecret))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Basic "+credentials)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get access token: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResponse TokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return "", err
	}

	return tokenResponse.AccessToken, nil
}

// alias payins
func StkPush(phoneNumber string, amount int, callbackURL, accountReference, consumerKey, consumerSecret, businessShortCode, passKey string) (*StkPushResponse, error) {
	url := "https://api.safaricom.co.ke/mpesa/stkpush/v1/processrequest"
	timestamp := time.Now().Format("20060102150405")
	token, err := GenerateAccessToken(consumerKey, consumerSecret)
	if err != nil {
		return nil, err
	}

	password := base64.StdEncoding.EncodeToString([]byte(businessShortCode + passKey + timestamp))

	requestBody := StkPushRequest{
		BusinessShortCode: businessShortCode,
		Password:          password,
		Timestamp:         timestamp,
		TransactionType:   "CustomerPayBillOnline",
		Amount:            amount,
		PartyA:            phoneNumber,
		PartyB:            businessShortCode,
		PhoneNumber:       phoneNumber,
		CallBackURL:       "https://payments.mam-laka.com/api/v1/mobile/callback",
		AccountReference:  accountReference,
		TransactionDesc:   "Payment",
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to initiate STK push: %s, %s", resp.Status, string(body))
	}

	// Parse the response body
	var stkResponse StkPushResponse
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &stkResponse)
	if err != nil {
		return nil, err
	}

	// Log or print the response for debugging
	fmt.Println("STK Push Response:", stkResponse)

	// Return the parsed response
	return &stkResponse, nil
}

func GenerateSecureID() string {
	b := make([]byte, 16) // Generate 16 random bytes
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

type B2CRequest struct {
	OriginatorConversationID string  `json:"OriginatorConversationID"`
	InitiatorName            string  `json:"InitiatorName"`
	SecurityCredential       string  `json:"SecurityCredential"`
	CommandID                string  `json:"CommandID"`
	Amount                   float64 `json:"Amount"`
	PartyA                   string  `json:"PartyA"`
	PartyB                   string  `json:"PartyB"`
	Remarks                  string  `json:"Remarks"`
	QueueTimeOutURL          string  `json:"QueueTimeOutURL"`
	ResultURL                string  `json:"ResultURL"`
	Occassion                string  `json:"Occassion"`
}

func generateB2BAccessToken(consumerKey, consumerSecret string) (string, error) {
	// Endpoint for generating the access token
	url := "https://api.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials"

	// Encode credentials in Base64
	credentials := base64.StdEncoding.EncodeToString([]byte(consumerKey + ":" + consumerSecret))

	// Create a new HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Add("Authorization", "Basic "+credentials)

	// Create HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse JSON response to extract the access token
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Extract and return the access token
	token, ok := response["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("access token not found in response")
	}

	return token, nil
}

func GenerateB2CRequest(phoneNumber string, amount float64, callbackURL, externalID string, identifier, consumerKey, consumerSecret, password, businessShortCode, initiatorName string) (*B2BResponse, error) {
	// consumer_key := "oLwt5LEkO7zkQaqV8Sy9Gs8MvgA8PFADM6VOUe4jYj98nVr1"
	// consumer_secret := "YylBuouNZdeOJeU8ltCKll5QBQ0xSDrdAq7pdaurpOS8FNYPkaSAA8kZLlblwslM"

	// consumerKey1 := "FYAzZv4GvPsYpIG0Yxan3k9llRAcv59HAnwP62pbr6gabOqf"
	// consumerSecret1 := "3IrR0Q0qbRnkhl2L2PB3oWDujrZpMvg00F7hYFBoihZGMpXuObCuKPzlFPIkJM2V"
	// initiatorName1 := "Collin"
	// password1 := "jUdSHSh84lzrYUnmIwfiZZIrOL7+o0sRRxteBLEJLO60lHVfV7K10ySoE0E8EqvbU6u6ZMNh6ATfQf8sU+XbFnWdMZUlADuhJXeUeGMk8Z842l8J8kWC3txYM1U0X5qDf3K/QnU26kj4UiRqhkXaIjJ69SL26ptVFozFYI2+8WXOH6Hhj20dDhWfsNaJCl8gYeAqdJMockmsZ1PQYNe6oph2jFPTS5kRKuXOglIYtVe97xkIdsnzKScseqTFRxm6Anlroi0fZLP9svNbOANSqTWY0p5rtuyILZlUD/gzWbAVlvO5SImLqI0RIikzAAuxnXvGkaKw36V795ItSwdeRQ=="
	// businessShortCode1 := "3008816"

	token, _ := generateB2BAccessToken(consumerKey, consumerSecret)
	fmt.Println("Access Token:", token)
	fmt.Printf("------Generating B2C request with phone: %s, amount: %.2f, callbackURL: %s, externalID: %s, identifier: %s, consumerKey: %s, consumerSecret: %s, password: %s, businessShortCode: %s, initiatorName: %s\n",
		RemovePlusPrefix(phoneNumber), amount, callbackURL, externalID, identifier, consumerKey, consumerSecret, password, businessShortCode, initiatorName)

	// businessShortCode := "3039805"
	// how is the password generated

	// password := "Xw8NWgC6K4Hnese1stlIMC0sE3p+kbcMtTVVxG57s4K/WZB2owiOf30B3yYSdTaTqdz2gv22we9sd4bgvfPVl7jynLtAglZn6KuGtdhhdy3eVQ0nosw3wZdfHDum8DCu5BAI/jU+x32PMSB/vtx9bbreV0rUHEvx7Gx4CI4Eze4BnhFQ368Z2x7x9Q+82r/tZxDlgG76NbWnLfj9DHbcs5hOBoMYiMbnXg8HsLUaI688qNGqqK9CLr8uKfIgXgFBSD4Ky7P9UwWBXlTOODtmv/TRJBnrD+8IFttZqjruDxV81NGIeASl9q6Ni8go5gBGrNHGxSJ/SF5rGhloTXLtHg=="

	re := regexp.MustCompile(`\D`)
	phoneNumberStr := re.ReplaceAllString(fmt.Sprintf("%s", phoneNumber), "")
	fmt.Println("Phone Number:", phoneNumberStr)
	fmt.Println("Identifier:", identifier)

	b2cRequest := B2CRequest{
		OriginatorConversationID: identifier,
		InitiatorName:            initiatorName,
		SecurityCredential:       password,
		CommandID:                "PromotionPayment",
		Amount:                   amount,
		PartyA:                   businessShortCode,
		PartyB:                   phoneNumberStr,
		Remarks:                  "payments done",
		QueueTimeOutURL:          "https://payments.mam-laka.com/api/v1/mobile/b2c/callback",
		ResultURL:                "https://payments.mam-laka.com/api/v1/mobile/b2c/callback",

		Occassion: "Ok",
	}
	// print the b2c request payload
	fmt.Println("B2C Request Payload:", b2cRequest)
	requestBody, err := json.Marshal(b2cRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request body: %w", err)
	}

	url := "https://api.safaricom.co.ke/mpesa/b2c/v1/paymentrequest"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	// Always print Safaricom's full JSON response
	fmt.Println("Safaricom Response Body:", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("B2C request failed with status %s: %s", resp.Status, string(body))
	}

	var b2bResponse B2BResponse
	err = json.Unmarshal(body, &b2bResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Safaricom JSON: %w. Raw body: %s", err, string(body))
	}

	fmt.Println("Parsed Response:", b2bResponse)
	return &b2bResponse, nil
}

// RemovePlusPrefix removes the '+' sign from the beginning of a phone number if present.
func RemovePlusPrefix(phoneNumber string) string {
	if len(phoneNumber) > 0 && phoneNumber[0] == '+' {
		return phoneNumber[1:] // Remove the first character
	}
	return phoneNumber
}
