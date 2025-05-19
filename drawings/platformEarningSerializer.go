package drawings

import "github.com/gin-gonic/gin"

type PlatformEarningSerializer struct {
	C        *gin.Context
	Earnings PlatformEarningModel
}

func NewPlatformEarningSerializer(c *gin.Context, earning PlatformEarningModel) PlatformEarningSerializer {
	return PlatformEarningSerializer{C: c, Earnings: earning}
}

func (s PlatformEarningSerializer) Response() map[string]interface{} {
	return map[string]interface{}{
		"id":                 s.Earnings.ID,
		"impalaMerchantId":   s.Earnings.ImpalaMerchantID,
		"amountTransferred":  s.Earnings.AmountTransferred,
		"transactionCharges": s.Earnings.TransactionCharges,
		"transferDate":       s.Earnings.TransferDate.Format("2006-01-02 15:04:05"),
	}
}
