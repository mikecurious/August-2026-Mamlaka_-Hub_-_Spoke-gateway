package merchants

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Function to send a token transfer
func sendTokenTransfer(amount, receiverAddress string) {
	fmt.Println("am here ... for the token ")
	// url := "127.0.0.1:5000/tokenTransfer"
	url := "https://chain.impalapay.com/tokenTransfer" // Make sure to include 'http://' in the URL
	method := "POST"

	// Define the constant fields
	assetCode := "IMC"
	issuerAddress := "GBKXT3O3JDSPLM36XQJG3E72QELD3UJRVVMPVKYAMV5O7N7BDFJ5CGRN"
	signerSeed := "SBYB3CETFLIX5RMGWB3AQHKDIM6LCMA4VU4W6YUO4GSDIA254QUW3VW2"

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
	req.Header.Add("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6Im1haW5haG13YW5naTEyQGdtYWlsLmNvbSIsImV4cCI6MTc0MTY3OTA3OSwicm9sZUlEIjoxLCJ1c2VySUQiOjd9.IdD4JsW-m5lYxlkh6ZWgVt6nQrRlKUk85RovnM4u3BM")

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
	signerSeed := "SABI4FZXS5LIYOUJJUPIYPIA3GJV6KGAUTGO6RNPP6IQOCKAFI2AEYJ5"

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
	req.Header.Add("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6Im1haW5haG13YW5naTEyQGdtYWlsLmNvbSIsImV4cCI6MTc0MTY3OTA3OSwicm9sZUlEIjoxLCJ1c2VySUQiOjd9.IdD4JsW-m5lYxlkh6ZWgVt6nQrRlKUk85RovnM4u3BM")

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
