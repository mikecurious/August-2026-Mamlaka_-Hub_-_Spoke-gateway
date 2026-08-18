package merchants

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// Environment variables holding the blockchain credentials. There are no
// fallbacks: an unset or blank value aborts the transfer rather than sending
// a request with a wrong or empty key.
//
//	BLOCKCHAIN_SIGNER_SEED_TRANSFER -- signer seed for sendTokenTransfer
//	BLOCKCHAIN_SIGNER_SEED_DEDUCT   -- signer seed for DeductTokenTransfer
//	CHAIN_API_BEARER_TOKEN          -- bearer token for chain.impalapay.com
const (
	envSignerSeedTransfer = "BLOCKCHAIN_SIGNER_SEED_TRANSFER"
	envSignerSeedDeduct   = "BLOCKCHAIN_SIGNER_SEED_DEDUCT"
	envChainBearerToken   = "CHAIN_API_BEARER_TOKEN"
)

// requireChainEnv reads one required blockchain credential. A missing or blank
// value is reported by name -- never by value -- and reported as not ok so the
// caller can abort before building a request.
func requireChainEnv(name string) (string, bool) {
	raw, ok := os.LookupEnv(name)
	if !ok || strings.TrimSpace(raw) == "" {
		log.Printf("token transfer aborted: required environment variable %s is unset or empty", name)
		return "", false
	}
	return raw, true
}

// Function to send a token transfer
func sendTokenTransfer(amount, receiverAddress string) {
	fmt.Println("am here ... for the token ")
	// url := "127.0.0.1:5000/tokenTransfer"
	url := "https://chain.impalapay.com/tokenTransfer" // Make sure to include 'http://' in the URL
	method := "POST"

	// Define the constant fields
	assetCode := "IMC"
	issuerAddress := "GBKXT3O3JDSPLM36XQJG3E72QELD3UJRVVMPVKYAMV5O7N7BDFJ5CGRN"
	signerSeed, ok := requireChainEnv(envSignerSeedTransfer)
	if !ok {
		return
	}
	bearerToken, ok := requireChainEnv(envChainBearerToken)
	if !ok {
		return
	}

	// Create the payload with dynamic amount and receiver address
	payload := strings.NewReader(fmt.Sprintf(`{
		"asset_code": "%s", 
		"issuer_address": "%s",
		"receiver_address": "%s",
		"signer_seed": "%s",
		"amount": "%s"                                                                                             
	}`, assetCode, issuerAddress, receiverAddress, signerSeed, amount))

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+bearerToken)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))
}

func DeductTokenTransfer(amount, Address string) {
	fmt.Println("am here for the token deduction")
	// url := "127.0.0.1:5000/tokenTransfer"
	url := "https://chain.impalapay.com/tokenTransfer" // Make sure to include 'http://' in the URL
	method := "POST"

	// Define the constant fields
	assetCode := "IMC"
	issuerAddress := "GBKXT3O3JDSPLM36XQJG3E72QELD3UJRVVMPVKYAMV5O7N7BDFJ5CGRN"
	signerSeed, ok := requireChainEnv(envSignerSeedDeduct)
	if !ok {
		return
	}
	bearerToken, ok := requireChainEnv(envChainBearerToken)
	if !ok {
		return
	}

	// Create the payload with dynamic amount and receiver address
	payload := strings.NewReader(fmt.Sprintf(`{
		"asset_code": "%s", 
		"issuer_address": "%s",
		"receiver_address": "%s",
		"signer_seed": "%s",
		"amount": "%s"                                                                                             
	}`, assetCode, issuerAddress, "GCPJDALEV53PNBYAFOU6XN4YU23AVJGWGJXVNSXTGE2ZVMILPH3EKK5V", signerSeed, amount))

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+bearerToken)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))
}

// func main() {
// 	// Example call to the function
// 	sendTokenTransfer("100", "GBR7COBB5T5WPEYI7PN2XXLIYC2VUE22VIF4BHRKA3TPLUSZS6TQY7BE")
// }
