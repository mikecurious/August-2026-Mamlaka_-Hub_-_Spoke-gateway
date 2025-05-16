package balances

// MerchantCollectionBalance represents the merchant balance data.
type MerchantCollectionBalance struct {
	ID               uint    `gorm:"primaryKey" json:"id"`
	ImpalaMerchantID string  `gorm:"column:impalaMerchantId;type:varchar(255);unique" json:"impalaMerchantId"`
	USDBalance       float64 `gorm:"column:usdBalance;type:float(100,2)" json:"usdBalance"`
	USDCBalance      float64 `gorm:"column:usdcBalance;type:float(100,7)" json:"usdcBalance"`
	ImpaBalance      float64 `gorm:"column:impaBalance;type:float(100,7)" json:"impaBalance"`
	LumenBalance     float64 `gorm:"column:lumenBalance;type:float(100,7)" json:"lumenBalance"`
	LastUpdated      int64   `gorm:"column:lastUpdated;type:int" json:"lastUpdated"`
	USDTBalance      float64 `gorm:"column:usdtBalance;type:float(100,2)" json:"usdtBalance"`
	KESBalance       float64 `gorm:"column:kesBalance;type:float(100,2)" json:"kesBalance"`
	EURBalance       float64 `gorm:"column:eurBalance;type:float(100,2)" json:"eurBalance"`
	GBPBalance       float64 `gorm:"column:gbpBalance;type:float(100,2)" json:"gbpBalance"`
	TZSBalance       float64 `gorm:"column:tzsBalance;type:float(100,2)" json:"tzsBalance"`
	UGXBalance       float64 `gorm:"column:ugxBalance;type:float(100,2)" json:"ugxBalance"`
	BaseCurrency     string  `gorm:"column:baseCurrency;type:varchar(3);default:USD" json:"baseCurrency"`
}

// TableName overrides the default table name.
func (MerchantCollectionBalance) TableName() string {
	return "merchant_collection_balance"
}
