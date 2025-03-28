package merchants

type STKResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
}

/*
 go run main.go
VgN8mZljJJg4ycWhVTKxeOrWEqRlResponse: {
            "ConversationID": "AG_20250201_206051c643ca6389833f",
            "OriginatorConversationID": "9021-4f51-8691-8604c07e13de14061111",
            "ResponseCode":"0",
            "ResponseDescription": "Accept the service request successfully."
        }

*/

type B2BResponse struct {
	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

//WITHdrawal sample request

/*
	{
	    "impalaMerchantId":"{{username}}",
	    "currency":"KES",
	    "amount":10,
	    "recipientPhone":"254112299271",
	    "mobileMoneySP":"M-Pesa",
	    "externalId":"joeltest4",
	    "callbackUrl":""
	}
*/
type MobileWithdrawalRequest struct {
	ImpalaMerchantId string  `json:"impalaMerchantId" binding:"required"`
	Currency         string  `json:"currency" binding:"required"`
	Amount           float32 `json:"amount" binding:"required"`
	RecipientPhone   string  `json:"recipientPhone" binding:"required"`
	MobileMoneySP    string  `json:"mobileMoneySP" binding:"required"`
	ExternalID       string  `json:"externalId" binding:"required"`
	CallbackURL      string  `json:"callbackUrl" binding:"required"`
}

// bank-pesa link request struct containing the followign, amount, destinationAccount, destinationBankCode, customerName string)
type PesalinkBankTransferRequest struct {
	Amount              float32 `json:"amount"`
	DestinationAccount  string  `json:"destinationAccount"`
	DestinationBankCode string  `json:"destinationBankCode"`
	CustomerName        string  `json:"customerName"`
}

// MobilePaymentRequest structure to bind incoming JSON request
type MobilePaymentRequest struct {
	ImpalaMerchantId string `json:"impalaMerchantId" binding:"required"`
	Currency         string `json:"currency" binding:"required"`
	Amount           int    `json:"amount" binding:"required"`
	DisplayName      string `json:"displayName" binding:"required"`
	PayerPhone       string `json:"payerPhone" binding:"required"`
	MobileMoneySP    string `json:"mobileMoneySP" binding:"required"`
	ExternalID       string `json:"externalId" binding:"required"`
	CallbackURL      string `json:"callbackUrl" binding:"required"`
}
type CardPaymentRequest struct {
	ImpalaMerchantId string  `json:"impalaMerchantId" binding:"required"`
	Currency         string  `json:"currency" binding:"required"`
	Amount           float32 `json:"amount" binding:"required"`
	MobileMoneySP    string  `json:"mobileMoneySP" binding:"required"`
	ExternalID       string  `json:"externalId" binding:"required"`
	CallbackURL      string  `json:"callbackUrl" binding:"required"`
	RedirectURL      string  `json:"redirectUrl"`
}

// to convert the merchant balances
type ConvertBalance struct {
	OriginCurrency      string  `json:"originCurrency" binding:"required"`
	DestinationCurrency string  `json:"destinationCurrency" binding:"required"`
	Amount              float64 `json:"amount" binding:"required"`
	ExchangeRate        float64 `json:"exchangeRate" binding:"required"`
}
