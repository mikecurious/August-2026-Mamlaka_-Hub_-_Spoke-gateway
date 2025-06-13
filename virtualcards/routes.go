package virtualcards

// import (
// 	"fmt"
// 	"net/http"
// 	"strconv"
// 	"time"

// 	"github.com/gin-gonic/gin"
// 	"github.com/golang-jwt/jwt/v4"
// )

// // Response structures for API responses
// type APIResponse struct {
// 	Success   bool        `json:"success"`
// 	Message   string      `json:"message"`
// 	Data      interface{} `json:"data,omitempty"`
// 	Error     string      `json:"error,omitempty"`
// 	Timestamp string      `json:"timestamp"`
// }

// type PaginatedResponse struct {
// 	Success    bool        `json:"success"`
// 	Message    string      `json:"message"`
// 	Data       interface{} `json:"data"`
// 	Pagination Pagination  `json:"pagination"`
// 	Timestamp  string      `json:"timestamp"`
// }

// type Pagination struct {
// 	CurrentPage int   `json:"current_page"`
// 	PerPage     int   `json:"per_page"`
// 	TotalPages  int   `json:"total_pages"`
// 	Total       int64 `json:"total"`
// }

// // Request structures

// type CreateVirtualCardRequest struct {
// 	CardTypeID  string  `json:"card_type_id" binding:"required"`
// 	HolderID    string  `json:"holder_id" binding:"required"`
// 	Amount      float64 `json:"amount" binding:"required,min=10"`
// 	Currency    string  `json:"currency" binding:"required"`
// 	Description string  `json:"description"`
// }

// type CardDepositRequest struct {
// 	CardID uint    `json:"card_id" binding:"required"`
// 	Amount float64 `json:"amount" binding:"required,min=10"`
// }

// type CardInfoRequest struct {
// 	CardID         uint `json:"card_id" binding:"required"`
// 	OnlySimpleInfo bool `json:"only_simple_info"`
// }

// // JWT Claims structure
// type Claims struct {
// 	UserID     uint   `json:"user_id"`
// 	MerchantID string `json:"merchant_id"`
// 	jwt.StandardClaims
// }

// // VirtualCardsHandler handles all virtual card related routes
// type VirtualCardsHandler struct {
// 	client *Client
// }

// // NewVirtualCardsHandler creates a new handler instance
// func NewVirtualCardsHandler(baseURL, apiKey string) *VirtualCardsHandler {
// 	return &VirtualCardsHandler{
// 		client: NewClient(baseURL, apiKey),
// 	}
// }

// // Middleware to extract merchant ID and user ID from JWT token
// func (h *VirtualCardsHandler) AuthMiddleware() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		tokenString := c.GetHeader("Authorization")
// 		if tokenString == "" {
// 			h.errorResponse(c, http.StatusUnauthorized, "Authorization header required", nil)
// 			c.Abort()
// 			return
// 		}

// 		// Remove "Bearer " prefix if present
// 		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
// 			tokenString = tokenString[7:]
// 		}

// 		// Parse the token
// 		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
// 			// Validate the signing method
// 			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
// 				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
// 			}
// 			return []byte("your-secret-key"), nil // Replace with your actual secret key
// 		})

// 		if err != nil {
// 			h.errorResponse(c, http.StatusUnauthorized, "Invalid token", err)
// 			c.Abort()
// 			return
// 		}

// 		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
// 			c.Set("user_id", claims.UserID)
// 			c.Set("merchant_id", claims.MerchantID)
// 		} else {
// 			h.errorResponse(c, http.StatusUnauthorized, "Invalid token claims", nil)
// 			c.Abort()
// 			return
// 		}

// 		c.Next()
// 	}
// }

// // SetupRoutes sets up all the virtual card routes
// func (h *VirtualCardsHandler) SetupRoutes(router *gin.Engine) {
// 	v1 := router.Group("/api/v1/virtual-cards")
// 	v1.Use(h.AuthMiddleware())

// 	// Card Holder routes
// 	v1.POST("/holders", h.CreateCardHolder)
// 	v1.GET("/holders", h.GetCardHolders)
// 	v1.GET("/holders/:id", h.GetCardHolder)
// 	v1.PUT("/holders/:id", h.UpdateCardHolder)
// 	v1.DELETE("/holders/:id", h.DeleteCardHolder)

// 	// Virtual Card routes
// 	v1.POST("/cards", h.CreateVirtualCard)
// 	v1.GET("/cards", h.GetVirtualCards)
// 	v1.GET("/cards/:id", h.GetVirtualCard)
// 	v1.PUT("/cards/:id", h.UpdateVirtualCard)
// 	v1.DELETE("/cards/:id", h.DeleteVirtualCard)
// 	v1.GET("/cards/:id/info", h.GetCardInfo)
// 	v1.GET("/cards/:id/balance", h.GetCardBalance)

// 	// Card Transaction routes
// 	v1.POST("/cards/:id/deposit", h.DepositToCard)
// 	v1.GET("/cards/:id/transactions", h.GetCardTransactions)
// 	v1.GET("/transactions", h.GetAllTransactions)
// 	v1.GET("/transactions/:id", h.GetTransaction)

// 	// Card Type routes
// 	v1.GET("/card-types", h.GetCardTypes)
// 	v1.GET("/card-types/active", h.GetActiveCardTypes)
// 	v1.GET("/card-types/:id", h.GetCardType)

// 	// Dashboard/Summary routes
// 	v1.GET("/summary", h.GetUserCardsSummary)
// 	v1.GET("/dashboard", h.GetDashboardData)
// }

// // CreateCardHolder creates a new card holder
// func (h *VirtualCardsHandler) CreateCardHolder(c *gin.Context) {
// 	var req CreateCardHolderRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		h.errorResponse(c, http.StatusBadRequest, "Invalid request data", err)
// 		return
// 	}

// 	userID := c.GetUint("user_id")
// 	merchantID := c.GetString("merchant_id")

// 	// Create API request
// 	apiReq := CreateHolderRequest{
// 		MerchantOrderNo: fmt.Sprintf("HOLDER-%s-%d", merchantID, time.Now().Unix()),
// 		CardTypeID:      req.CardTypeID,
// 		AreaCode:        req.AreaCode,
// 		Mobile:          req.Mobile,
// 		Email:           req.Email,
// 		FirstName:       req.FirstName,
// 		LastName:        req.LastName,
// 		BirthDay:        req.BirthDay,
// 		Country:         req.Country,
// 		Town:            req.Town,
// 		Address:         req.Address,
// 		PostCode:        req.PostCode,
// 	}

// 	// Call external API
// 	holderData, err := h.client.CreateHolder(apiReq)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusInternalServerError, "Failed to create holder via API", err)
// 		return
// 	}

// 	if !holderData.Success {
// 		h.errorResponse(c, http.StatusBadRequest, "Holder creation failed", fmt.Errorf(holderData.Message))
// 		return
// 	}

// 	// Save to database
// 	dbHolder := CardHolderModel{
// 		MerchantOrderNo: apiReq.MerchantOrderNo,
// 		HolderID:        holderData.HolderID,
// 		CardTypeID:      req.CardTypeID,
// 		AreaCode:        req.AreaCode,
// 		Mobile:          req.Mobile,
// 		Email:           req.Email,
// 		FirstName:       req.FirstName,
// 		LastName:        req.LastName,
// 		BirthDay:        req.BirthDay,
// 		Country:         req.Country,
// 		Town:            req.Town,
// 		Address:         req.Address,
// 		PostCode:        req.PostCode,
// 		Status:          holderData.Status,
// 		StatusStr:       holderData.StatusStr,
// 		Message:         holderData.Message2,
// 		UserID:          userID,
// 		MerchantID:      merchantID,
// 	}

// 	savedHolder, err := CreateCardHolderWithUser(userID, dbHolder)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusInternalServerError, "Failed to save holder to database", err)
// 		return
// 	}

// 	h.successResponse(c, http.StatusCreated, "Card holder created successfully", savedHolder)
// }

// // GetCardHolders retrieves all card holders for the authenticated user
// func (h *VirtualCardsHandler) GetCardHolders(c *gin.Context) {
// 	userID := c.GetUint("user_id")

// 	holders, err := GetCardHoldersByUserID(userID)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve card holders", err)
// 		return
// 	}

// 	h.successResponse(c, http.StatusOK, "Card holders retrieved successfully", holders)
// }

// // GetCardHolder retrieves a specific card holder
// func (h *VirtualCardsHandler) GetCardHolder(c *gin.Context) {
// 	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusBadRequest, "Invalid holder ID", err)
// 		return
// 	}

// 	userID := c.GetUint("user_id")

// 	holder, err := GetCardHolderByID(uint(id))
// 	if err != nil {
// 		h.errorResponse(c, http.StatusNotFound, "Card holder not found", err)
// 		return
// 	}

// 	// Check if the holder belongs to the authenticated user
// 	if holder.UserID != userID {
// 		h.errorResponse(c, http.StatusForbidden, "Access denied", nil)
// 		return
// 	}

// 	h.successResponse(c, http.StatusOK, "Card holder retrieved successfully", holder)
// }

// // UpdateCardHolder updates a card holder
// func (h *VirtualCardsHandler) UpdateCardHolder(c *gin.Context) {
// 	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusBadRequest, "Invalid holder ID", err)
// 		return
// 	}

// 	var req CreateCardHolderRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		h.errorResponse(c, http.StatusBadRequest, "Invalid request data", err)
// 		return
// 	}

// 	userID := c.GetUint("user_id")

// 	holder, err := GetCardHolderByID(uint(id))
// 	if err != nil {
// 		h.errorResponse(c, http.StatusNotFound, "Card holder not found", err)
// 		return
// 	}

// 	// Check if the holder belongs to the authenticated user
// 	if holder.UserID != userID {
// 		h.errorResponse(c, http.StatusForbidden, "Access denied", nil)
// 		return
// 	}

// 	// Update holder data
// 	updateData := map[string]interface{}{
// 		"card_type_id": req.CardTypeID,
// 		"area_code":    req.AreaCode,
// 		"mobile":       req.Mobile,
// 		"email":        req.Email,
// 		"first_name":   req.FirstName,
// 		"last_name":    req.LastName,
// 		"birth_day":    req.BirthDay,
// 		"country":      req.Country,
// 		"town":         req.Town,
// 		"address":      req.Address,
// 		"post_code":    req.PostCode,
// 	}

// 	err = UpdateSingleCardHolder(&holder, updateData)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusInternalServerError, "Failed to update card holder", err)
// 		return
// 	}

// 	h.successResponse(c, http.StatusOK, "Card holder updated successfully", holder)
// }

// // DeleteCardHolder deletes a card holder
// func (h *VirtualCardsHandler) DeleteCardHolder(c *gin.Context) {
// 	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusBadRequest, "Invalid holder ID", err)
// 		return
// 	}

// 	userID := c.GetUint("user_id")

// 	holder, err := GetCardHolderByID(uint(id))
// 	if err != nil {
// 		h.errorResponse(c, http.StatusNotFound, "Card holder not found", err)
// 		return
// 	}

// 	// Check if the holder belongs to the authenticated user
// 	if holder.UserID != userID {
// 		h.errorResponse(c, http.StatusForbidden, "Access denied", nil)
// 		return
// 	}

// 	err = DeleteSingleCardHolder(&holder)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusInternalServerError, "Failed to delete card holder", err)
// 		return
// 	}

// 	h.successResponse(c, http.StatusOK, "Card holder deleted successfully", nil)
// }

// // CreateVirtualCard creates a new virtual card
// func (h *VirtualCardsHandler) CreateVirtualCard(c *gin.Context) {
// 	var req CreateVirtualCardRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		h.errorResponse(c, http.StatusBadRequest, "Invalid request data", err)
// 		return
// 	}

// 	userID := c.GetUint("user_id")
// 	merchantID := c.GetString("merchant_id")

// 	// Verify holder exists and belongs to user
// 	holder, err := GetCardHolderByHolderID(req.HolderID)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusNotFound, "Card holder not found", err)
// 		return
// 	}

// 	if holder.UserID != userID {
// 		h.errorResponse(c, http.StatusForbidden, "Access denied to holder", nil)
// 		return
// 	}

// 	// Create API request
// 	apiReq := CreateCardRequest{
// 		CardTypeID: req.CardTypeID,
// 		HolderID:   req.HolderID,
// 		Amount:     int(req.Amount),
// 	}

// 	// Call external API
// 	cardData, err := h.client.CreateCard(apiReq)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusInternalServerError, "Failed to create card via API", err)
// 		return
// 	}

// 	// Save to database
// 	dbCard := VirtualCardModel{
// 		OrderNo:          cardData.OrderNo,
// 		MerchantOrderNo:  cardData.MerchantOrderNo,
// 		CardTypeID:       req.CardTypeID,
// 		HolderID:         req.HolderID,
// 		CardNo:           cardData.OrderNo, // Using OrderNo as CardNo initially
// 		Currency:         cardData.Currency,
// 		Amount:           cardData.Amount,
// 		Fee:              cardData.Fee,
// 		ReceivedAmount:   cardData.ReceivedAmount,
// 		ReceivedCurrency: cardData.ReceivedCurrency,
// 		Type:             cardData.Type,
// 		Status:           cardData.Status,
// 		TransactionTime:  cardData.TransactionTime,
// 		Balance:          req.Amount,
// 		UserID:           userID,
// 		CardHolderID:     holder.ID,
// 		MerchantID:       merchantID,
// 	}

// 	savedCard, err := CreateVirtualCardWithTransaction(userID, dbCard)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusInternalServerError, "Failed to save card to database", err)
// 		return
// 	}

// 	h.successResponse(c, http.StatusCreated, "Virtual card created successfully", savedCard)
// }

// // GetVirtualCards retrieves all virtual cards for the authenticated user
// func (h *VirtualCardsHandler) GetVirtualCards(c *gin.Context) {
// 	userID := c.GetUint("user_id")

// 	cards, err := GetVirtualCardsByUserID(userID)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve virtual cards", err)
// 		return
// 	}

// 	h.successResponse(c, http.StatusOK, "Virtual cards retrieved successfully", cards)
// }

// // GetVirtualCard retrieves a specific virtual card
// func (h *VirtualCardsHandler) GetVirtualCard(c *gin.Context) {
// 	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusBadRequest, "Invalid card ID", err)
// 		return
// 	}

// 	userID := c.GetUint("user_id")

// 	card, err := GetVirtualCardByID(uint(id))
// 	if err != nil {
// 		h.errorResponse(c, http.StatusNotFound, "Virtual card not found", err)
// 		return
// 	}

// 	// Check if the card belongs to the authenticated user
// 	if card.UserID != userID {
// 		h.errorResponse(c, http.StatusForbidden, "Access denied", nil)
// 		return
// 	}

// 	h.successResponse(c, http.StatusOK, "Virtual card retrieved successfully", card)
// }

// // UpdateVirtualCard updates a virtual card
// func (h *VirtualCardsHandler) UpdateVirtualCard(c *gin.Context) {
// 	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusBadRequest, "Invalid card ID", err)
// 		return
// 	}

// 	userID := c.GetUint("user_id")

// 	card, err := GetVirtualCardByID(uint(id))
// 	if err != nil {
// 		h.errorResponse(c, http.StatusNotFound, "Virtual card not found", err)
// 		return
// 	}

// 	// Check if the card belongs to the authenticated user
// 	if card.UserID != userID {
// 		h.errorResponse(c, http.StatusForbidden, "Access denied", nil)
// 		return
// 	}

// 	var updateData map[string]interface{}
// 	if err := c.ShouldBindJSON(&updateData); err != nil {
// 		h.errorResponse(c, http.StatusBadRequest, "Invalid request data", err)
// 		return
// 	}

// 	err = UpdateSingleVirtualCard(&card, updateData)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusInternalServerError, "Failed to update virtual card", err)
// 		return
// 	}

// 	h.successResponse(c, http.StatusOK, "Virtual card updated successfully", card)
// }

// // DeleteVirtualCard deletes a virtual card
// func (h *VirtualCardsHandler) DeleteVirtualCard(c *gin.Context) {
// 	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusBadRequest, "Invalid card ID", err)
// 		return
// 	}

// 	userID := c.GetUint("user_id")

// 	card, err := GetVirtualCardByID(uint(id))
// 	if err != nil {
// 		h.errorResponse(c, http.StatusNotFound, "Virtual card not found", err)
// 		return
// 	}

// 	// Check if the card belongs to the authenticated user
// 	if card.UserID != userID {
// 		h.errorResponse(c, http.StatusForbidden, "Access denied", nil)
// 		return
// 	}

// 	err = DeleteSingleVirtualCard(&card)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusInternalServerError, "Failed to delete virtual card", err)
// 		return
// 	}

// 	h.successResponse(c, http.StatusOK, "Virtual card deleted successfully", nil)
// }

// // GetCardInfo retrieves detailed card information
// func (h *VirtualCardsHandler) GetCardInfo(c *gin.Context) {
// 	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusBadRequest, "Invalid card ID", err)
// 		return
// 	}

// 	userID := c.GetUint("user_id")

// 	card, err := GetVirtualCardByID(uint(id))
// 	if err != nil {
// 		h.errorResponse(c, http.StatusNotFound, "Virtual card not found", err)
// 		return
// 	}

// 	// Check if the card belongs to the authenticated user
// 	if card.UserID != userID {
// 		h.errorResponse(c, http.StatusForbidden, "Access denied", nil)
// 		return
// 	}

// 	// Get card info from external API
// 	apiReq := CardInfoRequest{
// 		CardNo:         card.CardNo,
// 		OnlySimpleInfo: true,
// 	}

// 	cardInfo, err := h.client.GetCardInfo(apiReq)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve card info from API", err)
// 		return
// 	}

// 	// Combine database and API data
// 	response := map[string]interface{}{
// 		"database_info": card,
// 		"api_info":      cardInfo,
// 	}

// 	h.successResponse(c, http.StatusOK, "Card information retrieved successfully", response)
// }

// // GetCardBalance retrieves the current balance of a card
// func (h *VirtualCardsHandler) GetCardBalance(c *gin.Context) {
// 	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusBadRequest, "Invalid card ID", err)
// 		return
// 	}

// 	userID := c.GetUint("user_id")

// 	card, err := GetVirtualCardByID(uint(id))
// 	if err != nil {
// 		h.errorResponse(c, http.StatusNotFound, "Virtual card not found", err)
// 		return
// 	}

// 	// Check if the card belongs to the authenticated user
// 	if card.UserID != userID {
// 		h.errorResponse(c, http.StatusForbidden, "Access denied", nil)
// 		return
// 	}

// 	balance, err := GetCardBalance(uint(id))
// 	if err != nil {
// 		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve card balance", err)
// 		return
// 	}

// 	response := map[string]interface{}{
// 		"card_id":  id,
// 		"balance":  balance,
// 		"currency": card.Currency,
// 	}

// 	h.successResponse(c, http.StatusOK, "Card balance retrieved successfully", response)
// }

// // DepositToCard processes a deposit to a virtual card
// func (h *VirtualCardsHandler) DepositToCard(c *gin.Context) {
// 	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusBadRequest, "Invalid card ID", err)
// 		return
// 	}

// 	var req CardDepositRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		h.errorResponse(c, http.StatusBadRequest, "Invalid request data", err)
// 		return
// 	}

// 	userID := c.GetUint("user_id")
// 	merchantID := c.GetString("merchant_id")

// 	card, err := GetVirtualCardByID(uint(id))
// 	if err != nil {
// 		h.errorResponse(c, http.StatusNotFound, "Virtual card not found", err)
// 		return
// 	}

// 	// Check if the card belongs to the authenticated user
// 	if card.UserID != userID {
// 		h.errorResponse(c, http.StatusForbidden, "Access denied", nil)
// 		return
// 	}

// 	// Process deposit
// 	orderNo := fmt.Sprintf("DEP-%s-%d", merchantID, time.Now().Unix())
// 	merchantOrderNo := fmt.Sprintf("MDEP-%s-%d", merchantID, time.Now().Unix())

// 	err = ProcessCardDeposit(uint(id), userID, req.Amount, orderNo, merchantOrderNo)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusInternalServerError, "Failed to process deposit", err)
// 		return
// 	}

// 	// Also call external API for deposit
// 	apiReq := DepositRequest{
// 		CardNo: card.CardNo,
// 		Amount: req.Amount,
// 	}

// 	_, err = h.client.DepositToCard(apiReq)
// 	if err != nil {
// 		// Log the error but don't fail the request since we've already updated our database
// 		fmt.Printf("Warning: Failed to sync deposit with external API: %v\n", err)
// 	}

// 	h.successResponse(c, http.StatusOK, "Deposit processed successfully", map[string]interface{}{
// 		"amount":       req.Amount,
// 		"order_no":     orderNo,
// 		"card_id":      id,
// 		"processed_at": time.Now(),
// 	})
// }

// // GetCardTransactions retrieves transactions for a specific card
// func (h *VirtualCardsHandler) GetCardTransactions(c *gin.Context) {
// 	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusBadRequest, "Invalid card ID", err)
// 		return
// 	}

// 	userID := c.GetUint("user_id")

// 	card, err := GetVirtualCardByID(uint(id))
// 	if err != nil {
// 		h.errorResponse(c, http.StatusNotFound, "Virtual card not found", err)
// 		return
// 	}

// 	// Check if the card belongs to the authenticated user
// 	if card.UserID != userID {
// 		h.errorResponse(c, http.StatusForbidden, "Access denied", nil)
// 		return
// 	}

// 	transactions, err := GetTransactionsByCardID(uint(id))
// 	if err != nil {
// 		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve transactions", err)
// 		return
// 	}

// 	h.successResponse(c, http.StatusOK, "Card transactions retrieved successfully", transactions)
// }

// // GetAllTransactions retrieves all transactions for the authenticated user
// func (h *VirtualCardsHandler) GetAllTransactions(c *gin.Context) {
// 	userID := c.GetUint("user_id")

// 	transactions, err := GetTransactionsByUserID(userID)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve transactions", err)
// 		return
// 	}

// 	h.successResponse(c, http.StatusOK, "Transactions retrieved successfully", transactions)
// }

// // GetTransaction retrieves a specific transaction
// func (h *VirtualCardsHandler) GetTransaction(c *gin.Context) {
// 	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusBadRequest, "Invalid transaction ID", err)
// 		return
// 	}

// 	userID := c.GetUint("user_id")

// 	transaction, err := GetCardTransactionByID(uint(id))
// 	if err != nil {
// 		h.errorResponse(c, http.StatusNotFound, "Transaction not found", err)
// 		return
// 	}

// 	// Check if the transaction belongs to the authenticated user
// 	if transaction.UserID != userID {
// 		h.errorResponse(c, http.StatusForbidden, "Access denied", nil)
// 		return
// 	}

// 	h.successResponse(c, http.StatusOK, "Transaction retrieved successfully", transaction)
// }

// // GetCardTypes retrieves all card types
// func (h *VirtualCardsHandler) GetCardTypes(c *gin.Context) {
// 	cardTypes, err := GetAllCardTypes()
// 	if err != nil {
// 		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve card types", err)
// 		return
// 	}

// 	h.successResponse(c, http.StatusOK, "Card types retrieved successfully", cardTypes)
// }

// // GetActiveCardTypes retrieves all active card types
// func (h *VirtualCardsHandler) GetActiveCardTypes(c *gin.Context) {
// 	cardTypes, err := GetActiveCardTypes()
// 	if err != nil {
// 		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve active card types", err)
// 		return
// 	}

// 	h.successResponse(c, http.StatusOK, "Active card types retrieved successfully", cardTypes)
// }

// // GetCardType retrieves a specific card type
// func (h *VirtualCardsHandler) GetCardType(c *gin.Context) {
// 	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusBadRequest, "Invalid card type ID", err)
// 		return
// 	}

// 	cardType, err := GetCardTypeByID(uint(id))
// 	if err != nil {
// 		h.errorResponse(c, http.StatusNotFound, "Card type not found", err)
// 		return
// 	}

// 	h.successResponse(c, http.StatusOK, "Card type retrieved successfully", cardType)
// }

// // GetUserCardsSummary retrieves a summary of user's cards
// func (h *VirtualCardsHandler) GetUserCardsSummary(c *gin.Context) {
// 	userID := c.GetUint("user_id")

// 	summary, err := GetUserCardsSummary(userID)
// 	if err != nil {
// 		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve cards summary", err)
// 		return
// 	}

// 	h.successResponse(c, http.StatusOK, "Cards summary retrieved successfully", summary)
// }

// // GetDashboardData retrieves comprehensive dashboard data
// // func (h *VirtualCardsHandler) GetDashboardData(c *gin.Context) {
// // 	userID := c.GetUint("user_id")

// // 	// Get summary
// // 	summary, err := GetUserCardsSummary(userID)
// // 	if err != nil {
// // 		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve dashboard data", err)
// // 		return
// // 	}

// // 	// Get recent transactions
// // 	transactions, err := GetTransactionsByUserID(userID)
// // 	if err != nil {
// // 		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve recent transactions", err)
// // 		return
// // 	}

// // 	// Limit to recent 10 transactions
// // 	recent
