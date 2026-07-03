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
	// ConsumerKey       = "a53D2lxIgGTgXtTDEnMo5btDnG90nOhg16GOK0MAlOOQNhBe"
	// ConsumerSecret    = "LchqRZnB48pQfGB1WUvNhp6qqzGQ3MfBFd32sGsqvYIzvmJswghXXWA0KormP3NV"
	// BusinessShortCode = "4130455"
	// PassKey           = "172f9892373eafe6dac71a87e4e8ade1792599809f7de1667c647bce03364ca7"
	// Password          = "Xw8NWgC6K4Hnese1stlIMC0sE3p+kbcMtTVVxG57s4K/WZB2owiOf30B3yYSdTaTqdz2gv22we9sd4bgvfPVl7jynLtAglZn6KuGtdhhdy3eVQ0nosw3wZdfHDum8DCu5BAI/jU+x32PMSB/vtx9bbreV0rUHEvx7Gx4CI4Eze4BnhFQ368Z2x7x9Q+82r/tZxDlgG76NbWnLfj9DHbcs5hOBoMYiMbnXg8HsLUaI688qNGqqK9CLr8uKfIgXgFBSD4Ky7P9UwWBXlTOODtmv/TRJBnrD+8IFttZqjruDxV81NGIeASl9q6Ni8go5gBGrNHGxSJ/SF5rGhloTXLtHg=="
	// InitiatorName     = "b2cInit"

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

	// b2c ...

	AppPayB2CConsumerKey    = "m99dbV8i4Vm4GgIn9yQ903a5kOZoi9DXClmFkVq4Aepo3ihx"
	AppPayB2CConsumerSecret = "x9CJvif8SpRg96qX0cUSyfyCrEkWjIjjwtoH5kGRIUE38orM714VImejkMDPs1EN"
	AppPayB2CInitiatorName  = "Collins"
	AppPayB2CPassword       = "fIbuUntzyiHE0Xb04RiDodF8paXQ2ujiZrG1uGtk7Mf5zEkE4h+Uqi9fYw2TscAZv0yCFL4bp83FtzRGOa5zJfPCMS5yojhOtg5hl4PL5FwHFW6kaeEqSK2rR0wZICUR9kQSQ+YAHxZkjiK33BF2Kh5AvIS1lmcp6rsQasQaCPF2o+YtXouiHTMPDHd3QpkywidN687/DKMIfIzA1K+bmBRXIb6kjd+Mu6dIpHBu78nzFHA6mUKtHNRAiOX+xTl7SdeLlmld/HLWbx10DBP1hozt6dx/4DCbX1nIef/4WtfWq8tKLCpTXVuuzTGc175baUhB2EcYJjeBPHoKnY5cLA=="
	AppPayB2CShortCode      = "3008818"
)

// Shiling-Bet m-pesa kenya
const (

	// 4904594
	ShilingiBetC2BConsumerKey       = "zhBLJbYmeE81THXKjOOQQld8V5HbxHwwKIGTZQAGlbhG6NK5"
	ShilingiBetC2BConsumerSecret    = "VD4MDNHkVT7kCcpcADD8rgrtmE4AJighFTHeNIywE4Cw39P36ptyIkJPSMqudEHp"
	ShilingiBetC2BBusinessShortCode = "4040811"
	ShilingiBetC2BPassKey           = "a8a2389145fe5219018b2c06d41971334f1806fb7a330800c253ecb49770c3ff"

	// b2c ...

	ShilingiBetB2CConsumerKey    = "m99dbV8i4Vm4GgIn9yQ903a5kOZoi9DXClmFkVq4Aepo3ihx"
	ShilingiBetB2CConsumerSecret = "x9CJvif8SpRg96qX0cUSyfyCrEkWjIjjwtoH5kGRIUE38orM714VImejkMDPs1EN"
	ShilingiBetB2CInitiatorName  = "Collins"
	ShilingiBetB2CPassword       = "Pka2rWhBsx3HdUpChiBUBotu47nXf6hoOZi7yNL+IO+hmewQ4v8segW/HjfRflylmIgRBfLD0NMJvtUASB7qDo7JRkHC/7jWAhUi3gJwaAV6X3yk5HNtwfpYm53wZcMqi6dOu1PH9Fj94Q0psg3DN6CiI3SZnxDNeWbeW5uIZPBQMTTOap04Wh0E4k9ygAgnCTXHOjMywQ3y5CgbfKwtvSnErOBzHtbGUvRoqOca66wkH5zdGA585OZtEjK2oJ/oYoxJuQ2K0iV4101Xa3RynS4XO2UOV9WalHXgKNi74E/vUCoSWHZJ80JpJ+myh6ficMuF3x3PB5xAqoN0lLyLmQ=="
	ShilingiBetB2CShortCode      = "3008818"
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
	CrayPayB2CPassword       = "H3y/unl9dwsviXb6RFQ20Fzdp3DBGWuFuec4tbVgCUQGFZeLVuOILMZLmzYTLGqRCXbxPmlou/VdYrLBwANoFbK53ZSdlW9DsLzWtcRSkrDEoiQU9mDpp4e9T8pPC1Jbg3rISAdTrOP72OBnZPZu5rBkIgMnBPnVa21TJfy3K3xY+Gta+txH4cbguoJ1/ffmhJmMqX0Gcr90N6ozTOWxVsTh4WE904YWxagJrK4iTvHBIAwQ07lnto2dlSMNYAiYwEJF4l5KoNa7v2gtsUr7b3VbQe+4TzQ4KE1N4BHMKIe/tJ7ml2QNn3USyK5gpcKT9zYX75gazfkfg4G3fw9QeA=="
	CrayPayB2CShortCode      = "3008814"
)

// TWD
const (
	TWDC2BConsumerKey       = "P2BwQcGj8fnnvsgirs2iFZ6pWbjLsrAJEzIlO2vvOnHM4uHS"
	TWDC2BConsumerSecret    = "26RPkoUxX4RRa9Wywg5RAQFOGwZwZYxjTzCIlO3kj6it56J9ze40TUxLvYMGj4Gd"
	TWDC2BBusinessShortCode = "4041809"
	TWDC2BPassKey           = "ef99e90e1744e1a689df2a3c2bcee521caf8c93d7db8f723c7634ad8424db324"

	TWDPayB2CConsumerKey    = "AG3ayMSMz6Se4JPkdC7h7Z1yOAMHlAzGLuicI3GfWP2cijmO"
	TWDPayB2CConsumerSecret = "kJIJRzMie4HbljhhF9d9d7bEvMJkMMjPoA4AHLyThrSSxDO8w2uFQUzY3AhI0Ey4"
	TWDPayB2CInitiatorName  = "collins"
	TWDPayB2CPassword       = "kaEiK3aDSUdZrHOr2dHsN6YgRAd9f3eYl02E4xUuZ7Gbjv6mAa7G8BNgxCYQaR1JCiqydFa5ksFRc+K5Agg+vQFFwcbBUCQHm5N0ZaXUoVonlQ3Z9aqQJObnHgpNQbUq5GpXPENJZSsr2rNb4ZHIKeJfXX+kmw3hNYiePQUmaIKDt5+Py/60GcfWzbaUgQkGqI1yefgSe/H95Kuha2TX/g5nbD4U0cyko1m8aneeMV8asAnnCYlMk+GzCPRcEf1gsIC2pU9KXBAqIvHoXxz8wRaaMENQSy39+OO03kb5zV7L36nWpLhecJrPL5YPzDdl/iYq+vj3LYpKfhTAH5AQlA=="
	TWDPayB2CShortCode      = "3008812"
)

// Lipad — Kenya M-Pesa
const (
	// Collection (C2B / STK paybill 4041887)
	LipadC2BConsumerKey        = "ITC9UqoLUF5iSGOIYH2fQYAGqQpLn1dJcsV2YKRRVbslI9DW"
	LipadC2BConsumerSecret     = "u9R2gmL2F5iklijz8PryuSmTY9oTdCVP1YHZ0lPd2gkmapCnfe7OLkM8r1gl2OGQ"
	LipadC2BBusinessShortCode  = "4041887"
	LipadC2BInitiatorName      = "Collins"
	LipadC2BPassKey            = "f79caa1b22f802af4f0489e03e2959d3d3463dc593e8bfe630adcac2b79d5c90"
	LipadC2BSecurityCredential = "VNTumMLkIpRAhV/ieQS1JxKVqWcCY1G3TcbnLRBCOkQqJJQuXyWH8sx2IM5/m8RojRr9A8w0tF/EJW0p5TXJ957IyPiLNFqtyA1SGQb7ikgosCpGZxtuO/+qNHt7a6uwV/d35/lAsw+cmQnSzNb36aDc23yZqgLis8qGU5QdluGO4QZT1QOsnlYCwnWSbsUhxjYdTxwahktLyyr3ShEckxAp5FTO5YG3s+HFctCjo44L0YmvomChlQSSw7/BD5/eb3rDQAUgQmnpYzpurBbNVrch9WaI0x+5d3eU83G7Ud8EhqYoDzdEWJoMkD54/w7oBvY0stIg9bTrhZQrzA4CEA=="

	// B2C payouts (shortcode 4564641)
	LipadPayB2CConsumerKey    = "OsUs5kpfSKdPOToCdY77ZB9dAE3HmX2zhwEfEVr1G66xOU1m"
	LipadPayB2CConsumerSecret = "PGNzG7TIZYP1LU8mNdSH5kvJx7GVD1ilfqW9xd9UnsBV5pIreONI0VebzqdsX5dg"
	LipadPayB2CInitiatorName  = "Collins"
	LipadPayB2CPassword       = "g/cA19za1LNCJXYCZbRLnfzq87iOfdL+tF3g17GTYc8nleCqm0vnchLdHnjr6mBgjIOZtaueJiOaM7xd0z6CnkESso437gtexhUMoShwVKKk9Kl+h4GsHiyQrKXrRDPVsyPLKtnvBZeKZXkgO2JmRU7pa1qRdjXXUYqVOJ2TO19mJpMh6FbqgGRTYCXwiwyaWyA+hxbMtjEu1nKFlE9Kk8TnEfBxCNpwOLVsTdkrK6siLbcUJ1NqE7MNEddIHKTF7z+4nOtEAPGTwXVneWvPeUnWL0ooqxdLis+81mLTxtxV2MisHjHHDgTWw/EuEGjG1PwtTE4grcPLTdKhj13M2Q=="
	LipadPayB2CShortCode      = "4564641"
)

// Neon — Kenya M-Pesa
const (
	// Collection (C2B / STK paybill 4041825)
	NeonC2BConsumerKey       = "Ne2drYwtVntB0COqNC2CrdUCPEK7FAPVpN07sOalaZkL4LYa"
	NeonC2BConsumerSecret    = "UfIQTtwSf0LjNhsz16ZsGkd2B1v9bAtPOstx4Haj1W1I9GMlc5aCpAfCCsxGlJn0"
	NeonC2BBusinessShortCode = "4041825"
	NeonC2BInitiatorName     = "Collins"
	NeonC2BPassKey           = "dc1f276a2ad324c4f5ed6418047ef4d44e0eb84b1921d38d40863d43f217564d"

	// B2C payouts (shortcode 4564637)
	NeonPayB2CConsumerKey    = "VlO2gxBwc7AANcBw3rBkSMyORAfD6fgGNiw6cnYk8YeyyNta"
	NeonPayB2CConsumerSecret = "GxFjK9Vlc75Az90xrJ5ehAIck5HNXqk4xls1KkhIcBXrETQYtBf5OFeCTZL3dSFA"
	NeonPayB2CInitiatorName  = "Collins"
	NeonPayB2CPassword       = "VNTumMLkIpRAhV/ieQS1JxKVqWcCY1G3TcbnLRBCOkQqJJQuXyWH8sx2IM5/m8RojRr9A8w0tF/EJW0p5TXJ957IyPiLNFqtyA1SGQb7ikgosCpGZxtuO/+qNHt7a6uwV/d35/lAsw+cmQnSzNb36aDc23yZqgLis8qGU5QdluGO4QZT1QOsnlYCwnWSbsUhxjYdTxwahktLyyr3ShEckxAp5FTO5YG3s+HFctCjo44L0YmvomChlQSSw7/BD5/eb3rDQAUgQmnpYzpurBbNVrch9WaI0x+5d3eU83G7Ud8EhqYoDzdEWJoMkD54/w7oBvY0stIg9bTrhZQrzA4CEA=="
	NeonPayB2CShortCode      = "4564637"
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
		TransactionDesc:   accountReference,
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
	return GenerateB2CRequestWithCommand(phoneNumber, amount, callbackURL, externalID, identifier, consumerKey, consumerSecret, password, businessShortCode, initiatorName, "PromotionPayment")
}

func GenerateB2CRequestWithCommand(phoneNumber string, amount float64, callbackURL, externalID string, identifier, consumerKey, consumerSecret, password, businessShortCode, initiatorName, commandID string) (*B2BResponse, error) {
	// consumer_key := "oLwt5LEkO7zkQaqV8Sy9Gs8MvgA8PFADM6VOUe4jYj98nVr1"
	// consumer_secret := "YylBuouNZdeOJeU8ltCKll5QBQ0xSDrdAq7pdaurpOS8FNYPkaSAA8kZLlblwslM"
	if commandID == "" {
		commandID = "PromotionPayment"
	}

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
		CommandID:                commandID,
		Amount:                   amount,
		PartyA:                   businessShortCode,
		PartyB:                   phoneNumberStr,
		Remarks:                  identifier,
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
