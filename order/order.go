package trade

import (
	"encoding/json"
	e "equity-trading/pkg/errors"
	"equity-trading/pkg/logger"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

/*
places normal order for product delivery, intraday with
segment equity, derivative, currency, commodity
with order type market, limit & validity ioc, intraday
*/
type trade struct{}
func (s *trade) PlaceOrder(c *gin.Context) {
	var (
		request  PlaceOrderRequest
		response PlaceOrderResponse
	)
	//  validating the request payload via gin framework
	if err := c.BindJSON(&request); err != nil {
		logger.Log.Error("Invalid arguement received", zap.Error(err))
		response.Errors = append(response.Errors, e.ErrorInfo["BadRequest"].GetErrorDetails(""))
		c.JSON(http.StatusBadRequest, response)
		c.Abort()
		return
	}
	// validating the request payload for trigger order
	if err := normalOrderValidation(request); err != nil {
		logger.Log.Error("normalOrderValidation Failed,", zap.Error(err))
		response.Errors = append(response.Errors, e.ErrorInfo["BadRequest"].GetErrorDetails(err.Error()))
		c.JSON(http.StatusBadRequest, response)
		c.Abort()
		return
	}

	st := time.Now()
	//creating rupeeseed api url for normal order
	uri := rupeeseedObj.EndPoint + OrderApi
	//creating request body for calling rupeeseed api
	requestBody := getRupeeseedOrderRequestBody(c, request)
	//call rupeeseed normalOrder api
	body, status, err := s.restCaller.InvokeHttp(http.MethodPost, uri, requestBody, rupeeseedHeaders, ApiTimeout) //calling the requestseed api with payload
	logger.Log.Info("api details", zap.Any("lateny", time.Since(st)), zap.Any("status", status), zap.Error(err), zap.Any("data", string(body)))
	if err != nil {
		// Rupeeseed api error handling
		logger.Log.Error("OrderEntry: rupeeseed api failure", zap.Error(err), zap.String("api:", uri))
		response.Errors = append(response.Errors, e.ErrorInfo["VendorApiFailure"].GetErrorDetails(""))
		c.JSON(http.StatusInternalServerError, response)
		c.Abort()
		return
	} else if status != http.StatusOK {
		// Rupeeseed api unable to place order
		logger.Log.Error("OrderEntry: api failure, failed to place order", zap.Error(err), zap.String("api:", uri))
		response.Errors = append(response.Errors, e.ErrorInfo["VendorConnectionFailure"].GetErrorDetails(""))
		c.JSON(status, response)
		c.Abort()
		return
	}

	// struct for parsing the rupeeseed api response
	var obj RupeeseedNormalOrderResponse
	// unmarshal rupeeseed response
	err = json.Unmarshal(body, &obj)
	// error occured while unmarshal the response
	if err != nil {
		logger.Log.Error("Failed to unmarshal rupeeseed output", zap.Error(err), zap.Any("recevied", string(body)))
		response.Errors = append(response.Errors, e.ErrorInfo["JsonUnmarshalError"].GetErrorDetails(""))
		c.JSON(http.StatusInternalServerError, response)
		c.Abort()
		return
	}
	// handling the failure of order placement at rupeeseed
	if obj.Status != Success {
		logger.Log.Error("order placement failed at rupeeseed", zap.String("msg", obj.Message), zap.String("errorCode", obj.ErrCode))
		if e.RupeeseedErrors[obj.ErrCode] == http.StatusBadRequest {
			response.Errors = append(response.Errors, e.ErrorInfo["BadRequest"].GetErrorDetails(fmt.Sprintf(":%s", obj.Message))) // Once error code recevies , need to handle errors
			c.JSON(http.StatusBadRequest, response)
		} else if e.RupeeseedErrors[obj.ErrCode] == http.StatusInternalServerError {
			response.Errors = append(response.Errors, e.ErrorInfo["InternalServerError"].GetErrorDetails(fmt.Sprintf(":%s", obj.Message)))
			c.JSON(http.StatusInternalServerError, response)
		} else {
			response.Errors = append(response.Errors, e.ErrorInfo["InternalServerError"].GetErrorDetails(""))
			c.JSON(http.StatusInternalServerError, response)
		}
		c.Abort()
		return
	}
	response.Data = obj.Data
	response.Status = true
	c.JSON(http.StatusOK, response)
}

/*
validating business constraints for placing
normal order
*/
func normalOrderValidation(request PlaceOrderRequest) error {
	//limit trigger order validation

	if request.OrderType == LMT && request.Price <= 0.0 {
		return errors.New(":Price cannot be zero with limit order")
	} else if request.OrderType == SL {
		if request.Price <= 0.0 {
			return errors.New(":Price cannot be zero with trigger limit order")
		}
		if request.TriggerPrice <= 0.0 {
			return errors.New(":Trigger Price cannot be zero with limit order")
		}
		if request.Validity == IOC {
			return errors.New(":Validity cannot be IOC with Trigger order") // NOTE this is SL/SLM orderType with Rupeeseed API
		}
		if request.TxnType == BUY && request.TriggerPrice > request.Price {
			return errors.New(":Trigger Price cannot be greater than limit buy price")
		}
		if request.TxnType == SELL && request.TriggerPrice < request.Price {
			return errors.New(":Trigger Price cannot be less than limit buy price")
		}
	} else if request.OrderType == SLM {
		if request.TriggerPrice <= 0.0 {
			return errors.New(":Trigger Price cannot be zero with limit order")
		} else if request.Validity == IOC {
			return errors.New(":Validity cannot be IOC with Trigger order") // NOTE this is SL/SLM orderType with Rupeeseed API
		}
	}

	if request.OffMktFlag && request.OffMktOrderTimeFlag <= 0 {
		return errors.New(":OffMktOrderTimeFlag possible allowed values are 1|2|3 in AMO")
	}

	return nil
}

/*
handles modification of normal order for product delivery, intraday with
segment equity, derivative, currency, commodity
with order type market, limit & validity ioc, intraday
*/
func (s *trade) ModifyOrder(c *gin.Context) {
	var (
		request  ModifyOrderRequest
		response Response
	)
	//  validating the request payload via gin framework
	if err := c.BindJSON(&request); err != nil {
		logger.Log.Error("Invalid arguement received", zap.Error(err))
		response.Errors = append(response.Errors, e.ErrorInfo["BadRequest"].GetErrorDetails(""))
		c.JSON(http.StatusBadRequest, response)
		c.Abort()
		return
	}

	//creating rupeeseed api url for normal order
	uri := rupeeseedObj.EndPoint + ModifyOrderApi
	//creating request body for calling rupeeseed api
	requestBody := parseVendorRequestBody(c, request)
	//call rupeeseed modifyNormalOrder api
	body, status, err := s.restCaller.InvokeHttp(http.MethodPost, uri, requestBody, rupeeseedHeaders, ApiTimeout)
	if err != nil {
		// Rupeeseed api error handling
		logger.Log.Error("OrderModify: rupeseed api failure", zap.Error(err), zap.String("api:", uri))
		response.Errors = append(response.Errors, e.ErrorInfo["VendorApiFailure"].GetErrorDetails(""))
		c.JSON(http.StatusInternalServerError, response)
		c.Abort()
		return
	} else if status != http.StatusOK {
		// Rupeeseed api unable to modify order
		logger.Log.Error("OrderEntry: api failure, failed to modify order", zap.Error(err), zap.String("api:", uri))
		response.Errors = append(response.Errors, e.ErrorInfo["VendorConnectionFailure"].GetErrorDetails(""))
		c.JSON(status, response)
		c.Abort()
		return
	}

	// struct for parsing the rupeeseed api response
	var obj RupeeseedNormalOrderResponse
	// unmarshal rupeeseed response
	err = json.Unmarshal(body, &obj)
	// error occured while unmarshal the response
	if err != nil {
		logger.Log.Error("Failed to marshal the error", zap.Error(err), zap.Any("recevied", string(body)))
		response.Errors = append(response.Errors, e.ErrorInfo["JsonUnmarshalError"].GetErrorDetails(""))
		c.JSON(http.StatusInternalServerError, response)
		c.Abort()
		return
	}
	// handling the failure of order modification at rupeeseed
	if obj.Status != Success {
		logger.Log.Error("order modify api failure at rupeeseed", zap.String("msg", obj.Message), zap.String("errorCode", obj.ErrCode))
		if e.RupeeseedErrors[obj.ErrCode] == http.StatusBadRequest {
			response.Errors = append(response.Errors, e.ErrorInfo["BadRequest"].GetErrorDetails(fmt.Sprintf(":%s", obj.Message))) // Once error code recevies , need to handle errors
			c.JSON(http.StatusBadRequest, response)
		} else if e.RupeeseedErrors[obj.ErrCode] == http.StatusInternalServerError {
			response.Errors = append(response.Errors, e.ErrorInfo["InternalServerError"].GetErrorDetails(fmt.Sprintf(":%s", obj.Message)))
			c.JSON(http.StatusInternalServerError, response)
		} else {
			response.Errors = append(response.Errors, e.ErrorInfo["InternalServerError"].GetErrorDetails(""))
			c.JSON(http.StatusInternalServerError, response)
		}
		c.Abort()
		return
	}
	response.Data = obj.Data
	response.Status = true
	response.Message = OrderModificationSuccess
	c.JSON(http.StatusOK, response)
}

/*
creating request body for calling rupeeseed api
for normal order through func PlaceOrder
*/
func getRupeeseedOrderRequestBody(c *gin.Context, req PlaceOrderRequest) RupeeseedNormalOrderRequest {
	temp := RupeeseedNormalOrderRequest{}
	//userId := c.GetString("userId")
	userId := "TEST2" // only for testing
	temp.EntityId = userId
	temp.Source = Source
	temp.Data.ClientId = userId
	temp.Data.UserId = userId
	temp.Data.TxnType = req.TxnType
	temp.Data.Exchange = req.Exchange
	temp.Data.Segment = req.Segment
	temp.Data.Product = req.Product
	temp.Data.ExchangeToken = fmt.Sprintf("%d", req.ExchangeToken)
	temp.Data.Qty = fmt.Sprintf("%d", req.Quantity)
	temp.Data.Price = fmt.Sprintf("%f", req.Price)
	temp.Data.Valdity = req.Validity
	temp.Data.OrderType = req.OrderType
	if req.DisclosedQty > 0 {
		temp.Data.DisclosedQty = fmt.Sprintf("%d", req.Quantity)
	}
	if req.TriggerPrice > 0.0 {
		temp.Data.TriggerPrice = fmt.Sprintf("%f", req.TriggerPrice)
	}
	temp.Data.OffMktFlag = "false"
	if req.OffMktFlag {
		temp.Data.OffMktFlag = "true"
		if req.OffMktOrderTimeFlag > 0 {
			temp.Data.EncashFlag = req.OffMktOrderTimeFlag
		}
	}

	return temp
}

/*
function prepares a request in Ruppeeseed API format from custom input request

input

	*gin.Context - for fetching client
	ConvertPositionRequest - cutom request to use to create Ruppeeseed API request

output

	RuppeeseedConvertPositionRequest - Ruppeeseed API request
*/
func getRupeeseedConvertPositionRequestBody(c *gin.Context, req ConvertPositionRequest) RuppeeseedConvertPositionRequest {
	temp := RuppeeseedConvertPositionRequest{}
	//userId := c.GetString("userId")
	userId := "TEST2" // only for testing
	temp.EntityID = userId
	temp.Source = Source
	temp.Data.ClientID = userId
	temp.Data.UserID = userId
	temp.Data.Exchange = req.Exchange
	temp.Data.SecurityID = req.ExchangeToken
	temp.Data.Segment = req.Segment
	temp.Data.Quantity = req.Quantity
	temp.Data.MktType = RuppeeSeedMarketType
	temp.Data.UserType = UserTypeClient
	temp.Data.TxnType = req.PositionType
	temp.Data.ProductFrom = req.PositionFrom
	temp.Data.ProductTo = req.PositionTo

	return temp
}

/*
OrderBook Returns details of orders placed by the user with
orders status - Pending, Partially Executed, Executed, Rejected, Cancelled
orders section - Open(pending, Partially Executed), Executed( Executed, Rejected, Cancelled)
*/
func (s *trade) OrderBook(c *gin.Context) {
	var (
		response OrderBookResponse
	)

	st := time.Now()
	//creating rupeeseed api url for OrderBook
	uri := rupeeseedObj.EndPoint + OrderBookApi
	//creating request body for calling rupeeseed OrderBook api
	requestBody := getOrderBookRupeeseedRequestBody(c)
	//call rupeeseed OrderBook api
	body, status, err := s.restCaller.InvokeHttp(http.MethodPost, uri, requestBody, rupeeseedHeaders, ApiTimeout)
	logger.Log.Info("api details", zap.Any("lateny", time.Since(st)), zap.Any("status", status), zap.Error(err), zap.Any("data", string(body)))
	if err != nil {
		// Rupeeseed api error handling
		logger.Log.Error("OrderBook: rupeeseed api failure", zap.Error(err), zap.String("api:", uri))
		response.Errors = append(response.Errors, e.ErrorInfo["VendorApiFailure"].GetErrorDetails(""))
		c.JSON(http.StatusInternalServerError, response)
		c.Abort()
		return
	} else if status != http.StatusOK {
		// Rupeeseed api unable to retreive orderBook i.e list of orders placed
		logger.Log.Error("OrderBook: rupeeseed api failure, failed to retreive orderBook", zap.Error(err), zap.String("api:", uri))
		response.Errors = append(response.Errors, e.ErrorInfo["VendorConnectionFailure"].GetErrorDetails(""))
		c.JSON(status, response)
		c.Abort()
		return
	}
	// struct for parsing the rupeeseed api response
	var obj RupeeseedOrderBookResponse
	// unmarshal rupeeseed response
	err = json.Unmarshal(body, &obj)
	// error occured while unmarshal the response
	if err != nil {
		logger.Log.Error("Failed to marshal the error", zap.Error(err), zap.Any("recevied", string(body)))
		response.Errors = append(response.Errors, e.ErrorInfo["JsonUnmarshalError"].GetErrorDetails(""))
		c.JSON(http.StatusInternalServerError, response)
		c.Abort()
		return
	}
	// handling the failure of OrderBook at rupeeseed
	if obj.Status != Success {
		logger.Log.Error("OrderBook api failure", zap.String("msg", obj.Message), zap.String("errorCode", obj.ErrorCode))
		if e.RupeeseedErrors[obj.ErrorCode] == http.StatusBadRequest {
			response.Errors = append(response.Errors, e.ErrorInfo["BadRequest"].GetErrorDetails(fmt.Sprintf(":%s", obj.Message))) // Once error code recevies , need to handle errors
			c.JSON(http.StatusBadRequest, response)
		} else if e.RupeeseedErrors[obj.ErrorCode] == http.StatusInternalServerError {
			response.Errors = append(response.Errors, e.ErrorInfo["InternalServerError"].GetErrorDetails(fmt.Sprintf(":%s", obj.Message)))
			c.JSON(http.StatusInternalServerError, response)
		} else {
			response.Errors = append(response.Errors, e.ErrorInfo["InternalServerError"].GetErrorDetails(""))
			c.JSON(http.StatusInternalServerError, response)
		}
		c.Abort()
		return
	}
	//creating list of struct OrderBook
	orderBookList := make([]OrderBook, 0)
	for _, rOrderBook := range obj.Data {
		var orderBook OrderBook

		if !filterOrder(c, rOrderBook) { //process only filtered order
			continue
		}

		orderBook.GoodTillDaysDate = rOrderBook.GoodTillDaysDate
		orderBook.Symbol = rOrderBook.Symbol
		orderBook.DqQtyRem = rOrderBook.DqQtyRem
		orderBook.DiscQuantity = rOrderBook.DiscQuantity
		orderBook.Price = rOrderBook.Price
		orderBook.Segment = rOrderBook.Segment
		orderBook.LotSize = rOrderBook.LotSize
		orderBook.OrderType = rOrderBook.OrderType
		orderBook.SecurityID = rOrderBook.SecurityID
		orderBook.ExpiryFlag = rOrderBook.ExpiryFlag
		orderBook.DisplayName = rOrderBook.DisplayName
		orderBook.ProductName = rOrderBook.ProductName
		orderBook.LastUpdatedTime = rOrderBook.LastUpdatedTime
		orderBook.TriggerPrice = rOrderBook.TriggerPrice
		orderBook.ExchOrderTime = rOrderBook.ExchOrderTime
		orderBook.Exchange = rOrderBook.Exchange
		orderBook.ErrorCode = rOrderBook.ErrorCode
		orderBook.SerialNo = rOrderBook.SerialNo
		orderBook.Status = rOrderBook.Status
		orderBook.OrderNo = rOrderBook.OrderNo
		orderBook.RemainingQuantity = rOrderBook.RemainingQuantity
		orderBook.ParticipantType = rOrderBook.ParticipantType
		orderBook.Product = rOrderBook.Product
		orderBook.OrderDateTime = rOrderBook.OrderDateTime
		orderBook.Quantity = rOrderBook.Quantity
		orderBook.ExpiryDate = rOrderBook.ExpiryDate
		orderBook.ExchOrderNo = rOrderBook.ExchOrderNo
		orderBook.TradedPrice = rOrderBook.TradedPrice
		orderBook.TxnType = rOrderBook.TxnType
		orderBook.RemQtyTotQty = rOrderBook.RemQtyTotQty
		orderBook.Validity = rOrderBook.Validity
		orderBook.AvgTradedPrice = rOrderBook.AvgTradedPrice
		orderBook.TradedQty = rOrderBook.TradedQty
		orderBook.StreamSymbol = rOrderBook.SecurityID + "_" + rOrderBook.Exchange

		/* Rupeeseed status | Rise Status 			| Section
		=========================================================
			Transit 		| Pending 			  	| Open
			Pending 		| Pending 			  	| Open
			Modified 		| Pending 			  	| Open
			Part-traded 	| Partially Executed 	| Open
			Traded 			| Executed 				| Executed
			Rejected 		| Rejected 				| Executed
			Cancelled 		| Cancelled 			| Executed
		*/
		if rOrderBook.Status == Traded || rOrderBook.Status == Rejected || rOrderBook.Status == Cancelled {
			if rOrderBook.Status == Traded {
				orderBook.Status = Executed
			}
			orderBook.Section = Executed
		}
		if rOrderBook.Status == Transit || rOrderBook.Status == Pending || rOrderBook.Status == Modified || rOrderBook.Status == PartTraded {
			if orderBook.Status == PartTraded {
				orderBook.Status = PartExecuted
			} else {
				orderBook.Status = Pending
			}
			orderBook.Section = Open
		}

		orderBookList = append(orderBookList, orderBook)
	}
	// if OrderBook is empty, no order placed
	if len(orderBookList) == 0 {
		logger.Log.Error("OrderBook api failure: order not found in OrderBook", zap.Error(err), zap.String("api:", uri))
		response.Errors = append(response.Errors, e.ErrorInfo["NoDataFound"].GetErrorDetails("order not found in OrderBook."))
		c.JSON(http.StatusNotFound, response)
		c.Abort()
		return
	}

	response.Data = orderBookList
	response.Status = true
	c.JSON(http.StatusOK, response)
}

/*
creating request body for calling rupeeseed api
for OrderBook through func OrderBook
*/
func getOrderBookRupeeseedRequestBody(c *gin.Context) RupeeseedOrderBookRequest {
	temp := RupeeseedOrderBookRequest{}
	userId := "TEST2" // only for testing
	//userId := c.GetString("userId")
	temp.EntityId = userId
	temp.Source = Source
	temp.Data.ClientId = userId
	temp.Data.UserId = userId

	return temp
}

/*
PositionBook Returns all open postions of segment
equity (T day’s trades, MTF trades), derivative
currency, commodity
*/
func (s *trade) PositionBook(c *gin.Context) {
	var (
		response PositionBookResponse
	)

	//Call rupeeseed PositionBook Api
	obj, err := s.fetchPositionBook(c)
	if err != nil {
		response.Errors = append(response.Errors, *err)
		if err.ErrName == e.BadRequest {
			c.JSON(http.StatusBadRequest, response)
		} else {
			c.JSON(http.StatusInternalServerError, response)
		}
		c.Abort()
		return
	}

	var totalPL float64

	positionBookList := make([]OrderPositionBook, 0)
	for _, rPosition := range obj.Data {

		var position OrderPositionBook
		position.Symbol = rPosition.Symbol
		position.NetQty = rPosition.NetQty
		position.SellAvg = rPosition.SellAvg
		position.BuyAvg = rPosition.BuyAvg
		position.GrossVal = rPosition.GrossVal
		position.TotSellQty = rPosition.TotSellQty
		position.LastTradedPrice = rPosition.LastTradedPrice
		position.Segment = rPosition.Segment
		position.GrossQty = rPosition.GrossQty
		position.LotSize = rPosition.LotSize
		position.TotSellVal = rPosition.TotSellVal
		position.Product = rPosition.Product
		position.NetAvg = rPosition.NetAvg
		position.ExpiryDate = rPosition.ExpiryDate
		position.TotSellValDay = rPosition.TotSellValDay
		position.TotBuyVal = rPosition.TotBuyVal
		position.NetVal = rPosition.NetVal
		position.DisplayName = rPosition.DisplayName
		position.TotBuyQty = rPosition.TotBuyQty
		position.Exchange = rPosition.Exchange
		position.SecurityID = rPosition.SecurityID
		position.StreamSymbol = rPosition.SecurityID + "_" + rPosition.Exchange
		position.RealisedProfit = rPosition.RealisedProfit

		var orderPL float64
		var unrealizedPL float64
		// if net quantity is +ve then it is a open buy position
		if rPosition.NetQty > 0 {
			// formula to calculate unrealised profit for open buy position
			// Total Qty*(LTP - Average Buy Price)
			unrealizedPL = float64(rPosition.NetQty) * (rPosition.LastTradedPrice - rPosition.BuyAvg)
			// if net quantity is -ve then it is a open sell position
		} else if rPosition.NetQty < 0 {
			//formula to calculate unrealised profit for open sell position
			//Total Quantity*(Average Sell Price - LTP)
			unrealizedPL = float64(rPosition.NetQty) * (rPosition.SellAvg - rPosition.LastTradedPrice)
		}
		// Total profit for a position =  unrealised profit + realised profit
		// Realised profit is for closed positions, value of realised profit, we are getting it from Rupeeseed Api
		// Unrealised profit is for open sell or open buy positions

		orderPL = unrealizedPL + rPosition.RealisedProfit

		// Total profit loss for all the positions in position book
		totalPL += orderPL

		positionBookList = append(positionBookList, position)
	}

	if len(positionBookList) == 0 {
		logger.Log.Error("PositionBook api failure: order not found in PositionBook", zap.Error(err))
		response.Errors = append(response.Errors, e.ErrorInfo["NoDataFound"].GetErrorDetails("order not found in PositionBook."))
		c.JSON(http.StatusNotFound, response)
		c.Abort()
		return
	}

	response.Data.OrderPosition = positionBookList
	response.Data.TotalProfitLoss = totalPL
	response.Status = true
	c.JSON(http.StatusOK, response)
}

/*
creating request body for calling rupeeseed api
for NetPosition through func PositionBook
*/
func getPositionBookRupeeseedRequestBody(c *gin.Context) RupeeseedPositionBookRequest {
	temp := RupeeseedPositionBookRequest{}
	userId := "TEST2" // only for testing
	//userId := c.GetString("userId")
	temp.EntityId = userId
	temp.Source = Source
	temp.Data.ClientId = userId
	temp.Data.UserId = userId
	temp.Data.InteropFlag = "IP"

	return temp
}

/*
creating request body for calling rupeeseed api
for modifying normal order, bracket order,
cover order through func ModifyOrder,BracketOrderModify, CoverOrderModify
*/
func parseVendorRequestBody(c *gin.Context, req ModifyOrderRequest) VendorRequest {
	temp := VendorRequest{}
	//userId := c.GetString("userId")
	userId := "TEST2" // only for testing
	temp.EntityId = userId
	temp.Source = Source
	temp.Data.ClientId = userId
	temp.Data.UserId = userId
	temp.Data.TxnType = req.TxnType
	temp.Data.Exchange = req.Exchange
	temp.Data.Segment = req.Segment
	temp.Data.Product = req.Product
	temp.Data.ExchangeToken = fmt.Sprintf("%d", req.ExchangeToken)
	temp.Data.Qty = fmt.Sprintf("%d", req.Qty)
	temp.Data.Price = fmt.Sprintf("%f", req.Price)
	temp.Data.Validity = req.Validity
	temp.Data.OrderType = req.OrderType
	temp.Data.DisclosedQty = fmt.Sprintf("%d", req.Qty)
	temp.Data.TriggerPrice = fmt.Sprintf("%f", req.TriggerPrice)
	temp.Data.OffMktFlag = fmt.Sprintf("%v", req.OffMktFlag)

	//For Order Modify API
	temp.Data.OrderNo = req.OrderNo
	temp.Data.GroupId = fmt.Sprintf("%d", req.GroupId)
	temp.Data.SerialNo = fmt.Sprintf("%d", req.SerialNo)

	//For Cover Order
	if req.Product == CoProductValue {
		temp.Data.LegNo = fmt.Sprintf("%f", req.LegNo)
	}

	//For Bracket Order
	if req.Product == CoProductValue {
		temp.Data.LegNo = fmt.Sprintf("%d", int(req.LegNo))
		temp.Data.AlgoOrderNo = req.AlgoOrderNo
	}
	return temp
}

func filterOrder(ctx *gin.Context, order RupeeseedOrderBook) bool {
	flag := true

	searchTxt := strings.ToLower(strings.TrimSpace(ctx.Query(SearchTxt)))
	segment := strings.ToLower(strings.TrimSpace(ctx.Query(Segment)))
	optionsType := strings.ToLower(strings.TrimSpace(ctx.Query(OptionsType)))
	status := strings.ToLower(strings.TrimSpace(ctx.Query(Status)))

	if len(searchTxt) != 0 {
		flag = flag && (strings.Contains(strings.ToLower(order.Symbol), searchTxt) ||
			strings.Contains(strings.ToLower(order.DisplayName), searchTxt))
	}

	if len(segment) != 0 {
		flag = flag && strings.ToLower(order.Segment) == segment
	}

	if len(optionsType) != 0 {
		flag = flag && strings.ToLower(order.OptType) == optionsType
	}

	if len(status) != 0 {
		orderStatus := strings.ToLower(order.Status)
		if status == strings.ToLower(Open) {
			flag = flag && (orderStatus == strings.ToLower(Transit) ||
				orderStatus == strings.ToLower(Pending) ||
				orderStatus == strings.ToLower(Modified) ||
				orderStatus == strings.ToLower(PartTraded))
		} else if status == strings.ToLower(Executed) {
			flag = flag && (orderStatus == strings.ToLower(Traded) ||
				orderStatus == strings.ToLower(Rejected) ||
				orderStatus == strings.ToLower(Cancelled))
		} else {
			flag = false //If status value is other than open/executed
		}

	}
	return flag
}

/*
places bracket order with product B(BO – Bracket Order)
segment E(equity), D(derivative)
with order type MKT(market), LMT(limit) &
validity IOC(Immediate or cance), DAY(Intraday)
*/
func (s *trade) PlaceBracketOrder(c *gin.Context) {
	var (
		request  PlaceBracketOrderRequest
		response PlaceBracketOrderResponse
	)
	//  validating the request payload via gin framework
	if err := c.BindJSON(&request); err != nil {
		logger.Log.Error("Invalid arguement received", zap.Error(err))
		response.Errors = append(response.Errors, e.ErrorInfo["BadRequest"].GetErrorDetails(""))
		c.JSON(http.StatusBadRequest, response)
		c.Abort()
		return
	}

	// validating the request payload for bracket order
	if err := bracketOrderValidation(request); err != nil {
		logger.Log.Error("bracketOrderValidation Failed,", zap.Error(err))
		response.Errors = append(response.Errors, e.ErrorInfo["BadRequest"].GetErrorDetails(err.Error()))
		c.JSON(http.StatusBadRequest, response)
		c.Abort()
		return
	}

	st := time.Now()
	//creating rupeeseed api url for BoOrderEntry
	uri := rupeeseedObj.EndPoint + BracketOrderApi
	//creating request body for calling rupeeseed api
	requestBody := getRupeseedBracketRequestBody(c, request)
	//call rupeeseed BoOrderEntry api
	body, status, err := s.restCaller.InvokeHttp(http.MethodPost, uri, requestBody, rupeeseedHeaders, ApiTimeout)
	logger.Log.Info("api details", zap.Any("lateny", time.Since(st)), zap.Any("status", status), zap.Error(err), zap.Any("data", string(body)))
	if err != nil {
		// Rupeeseed api error handling
		logger.Log.Error("BoOrderEntry: rupeeseed api failure", zap.Error(err), zap.String("api:", uri))
		response.Errors = append(response.Errors, e.ErrorInfo["VendorApiFailure"].GetErrorDetails(""))
		c.JSON(http.StatusInternalServerError, response)
		c.Abort()
		return
	} else if status != http.StatusOK {
		logger.Log.Error("BoOrderEntry: api failure, failed to place bracket order", zap.Error(err), zap.String("api:", uri))
		response.Errors = append(response.Errors, e.ErrorInfo["VendorConnectionFailure"].GetErrorDetails(""))
		c.JSON(status, response)
		c.Abort()
		return
	}

	// struct for parsing the rupeeseed api response
	var obj RupeeseedBracketOrderResponse
	// unmarshal rupeeseed response
	err = json.Unmarshal(body, &obj)
	// error occured while unmarshal the response
	if err != nil {
		logger.Log.Error("Failed to marshal the error", zap.Error(err), zap.Any("recevied", string(body)))
		response.Errors = append(response.Errors, e.ErrorInfo["JsonUnmarshalError"].GetErrorDetails(""))
		c.JSON(http.StatusInternalServerError, response)
		c.Abort()
		return
	}
	// handling the failure of Bracket order placement at rupeeseed
	if obj.Status != Success {
		logger.Log.Error("BoOrderEntry entry api failure", zap.String("msg", obj.Message), zap.String("errorCode", obj.ErrCode))
		if e.RupeeseedErrors[obj.ErrCode] == http.StatusBadRequest {
			response.Errors = append(response.Errors, e.ErrorInfo["BadRequest"].GetErrorDetails(fmt.Sprintf(":%s", obj.Message))) // Once error code recevies , need to handle errors
			c.JSON(http.StatusBadRequest, response)
		} else if e.RupeeseedErrors[obj.ErrCode] == http.StatusInternalServerError {
			response.Errors = append(response.Errors, e.ErrorInfo["InternalServerError"].GetErrorDetails(fmt.Sprintf(":%s", obj.Message)))
			c.JSON(http.StatusInternalServerError, response)
		} else {
			response.Errors = append(response.Errors, e.ErrorInfo["InternalServerError"].GetErrorDetails(""))
			c.JSON(http.StatusInternalServerError, response)
		}
		c.Abort()
		return
	}
	response.Data = obj.Data
	response.Status = true
	c.JSON(http.StatusOK, response)
}

/*
creating request body for calling rupeeseed api
for BoOrderEntry through func PlaceBracketOrder
*/
func getRupeseedBracketRequestBody(c *gin.Context, req PlaceBracketOrderRequest) RupeseedBracketOrderRequest {
	userId := "TEST2" // only for testing
	temp := RupeseedBracketOrderRequest{}
	//temp.EntityID = c.GetString("entityId")
	temp.EntityID = userId
	temp.Source = Source
	//temp.Data.ClientID = c.GetString("clientId")
	temp.Data.ClientID = userId
	temp.Data.TxnType = req.TxnType
	temp.Data.Exchange = req.Exchange
	temp.Data.Segment = req.Segment
	temp.Data.Product = req.Product
	temp.Data.Exchange = req.Exchange
	temp.Data.Quantity = fmt.Sprintf("%d", req.Quantity)
	temp.Data.Price = fmt.Sprintf("%f", req.Price)
	temp.Data.Validity = req.Validity
	temp.Data.OrderType = req.OrderType
	temp.Data.ProfitValue = fmt.Sprintf("%f", req.ProfitValue)
	temp.Data.StoplossValue = fmt.Sprintf("%f", req.StoplossValue)
	temp.Data.ExchangeToken = fmt.Sprintf("%d", req.ExchangeToken)
	if req.ProfitValue > 0.0 {
		temp.Data.ProfitValue = fmt.Sprintf("%f", req.ProfitValue)
	}
	if req.StoplossValue > 0.0 {
		temp.Data.StoplossValue = fmt.Sprintf("%f", req.StoplossValue)
	}
	if req.OffMktFlag {
		temp.Data.OffMktFlag = "true"
	} else {
		temp.Data.OffMktFlag = "false"
	}

	return temp
}

/*
places cover order with product V(CO – Cover Order)
segment E(equity), D(derivative)
with order type MKT(market), LMT(limit) &
validity IOC(Immediate or cancel), DAY(Intraday)
*/
func (s *trade) PlaceCoverOrder(c *gin.Context) {
	var (
		request  PlaceCoverOrderRequest
		response PlaceOrderResponse
	)
	//  validating the request payload via gin framework
	if err := c.BindJSON(&request); err != nil {
		logger.Log.Error("Invalid arguement received", zap.Error(err))
		response.Errors = append(response.Errors, e.ErrorInfo["BadRequest"].GetErrorDetails(""))
		c.JSON(http.StatusBadRequest, response)
		c.Abort()
		return
	}

	// validating the request payload for cover order
	if err := coverOrderValidation(request); err != nil {
		logger.Log.Error("coverOrderValidation Failed,", zap.Error(err))
		response.Errors = append(response.Errors, e.ErrorInfo["BadRequest"].GetErrorDetails(err.Error()))
		c.JSON(http.StatusBadRequest, response)
		c.Abort()
		return
	}

	st := time.Now()
	//creating rupeeseed api url for CoOrderEntry
	uri := rupeeseedObj.EndPoint + CoverOrderApi
	//creating request body for calling rupeeseed api
	requestBody := getRupeseedCoverRequestBody(c, request)
	//call rupeeseed CoOrderEntry api
	body, status, err := s.restCaller.InvokeHttp(http.MethodPost, uri, requestBody, rupeeseedHeaders, ApiTimeout)
	logger.Log.Info("api details", zap.Any("lateny", time.Since(st)), zap.Any("status", status), zap.Error(err), zap.Any("data", string(body)))
	if err != nil {
		// Rupeeseed api error handling
		logger.Log.Error("CoOrderEntry: rupeeseed api failure", zap.Error(err), zap.String("api:", uri))
		response.Errors = append(response.Errors, e.ErrorInfo["VendorApiFailure"].GetErrorDetails(""))
		c.JSON(http.StatusInternalServerError, response)
		c.Abort()
		return
	} else if status != http.StatusOK {
		logger.Log.Error("CoOrderEntry: rupeeseed api failure, failed to place bracket order", zap.Error(err), zap.String("api:", uri))
		response.Errors = append(response.Errors, e.ErrorInfo["VendorConnectionFailure"].GetErrorDetails(""))
		c.JSON(status, response)
		c.Abort()
		return
	}

	// struct for parsing the rupeeseed api response
	var obj RupeeseedNormalOrderResponse
	// unmarshal rupeeseed response
	err = json.Unmarshal(body, &obj)
	// error occured while unmarshal the response
	if err != nil {
		logger.Log.Error("Failed to marshal the error", zap.Error(err), zap.Any("recevied", string(body)))
		response.Errors = append(response.Errors, e.ErrorInfo["JsonUnmarshalError"].GetErrorDetails(""))
		c.JSON(http.StatusInternalServerError, response)
		c.Abort()
		return
	}
	// handling the failure of Cover order placement at rupeeseed
	if obj.Status != Success {
		logger.Log.Error("CoOrderEntry api failure", zap.String("msg", obj.Message), zap.String("errorCode", obj.ErrCode))
		if e.RupeeseedErrors[obj.ErrCode] == http.StatusBadRequest {
			response.Errors = append(response.Errors, e.ErrorInfo["BadRequest"].GetErrorDetails(fmt.Sprintf(":%s", obj.Message))) // Once error code recevies , need to handle errors
			c.JSON(http.StatusBadRequest, response)
		} else if e.RupeeseedErrors[obj.ErrCode] == http.StatusInternalServerError {
			response.Errors = append(response.Errors, e.ErrorInfo["InternalServerError"].GetErrorDetails(fmt.Sprintf(":%s", obj.Message)))
			c.JSON(http.StatusInternalServerError, response)
		} else {
			response.Errors = append(response.Errors, e.ErrorInfo["InternalServerError"].GetErrorDetails(""))
			c.JSON(http.StatusInternalServerError, response)
		}
		c.Abort()
		return
	}
	response.Data = obj.Data
	response.Status = true
	c.JSON(http.StatusOK, response)
}

/*
creating request body for calling rupeeseed api
for CoOrderEntry through func PlaceCoverOrder
*/
func getRupeseedCoverRequestBody(c *gin.Context, req PlaceCoverOrderRequest) RupeseedCoverOrderRequest {
	userId := "TEST2" // only for testing
	temp := RupeseedCoverOrderRequest{}
	//temp.EntityID = c.GetString("entityId")
	temp.EntityID = userId
	temp.Source = Source
	//temp.Data.ClientID = c.GetString("clientId")
	temp.Data.ClientID = userId
	temp.Data.TxnType = req.TxnType
	temp.Data.Exchange = req.Exchange
	temp.Data.Segment = req.Segment
	temp.Data.Product = req.Product
	temp.Data.Exchange = req.Exchange
	temp.Data.Quantity = fmt.Sprintf("%d", req.Quantity)
	temp.Data.Price = fmt.Sprintf("%f", req.Price)
	temp.Data.Validity = req.Validity
	temp.Data.OrderType = req.OrderType
	temp.Data.TriggerPrice = fmt.Sprintf("%f", req.TriggerPrice)
	temp.Data.ExchangeToken = fmt.Sprintf("%d", req.ExchangeToken)
	if req.OffMktFlag {
		temp.Data.OffMktFlag = "true"
	} else {
		temp.Data.OffMktFlag = "false"
	}

	return temp
}

/*
handles modification bracket order with product B(BO – Bracket Order)
segment E(equity), D(derivative)
with order type MKT(market), LMT(limit) &
validity IOC(Immediate or cance), DAY(Intraday)
*/
func (s *trade) BracketOrderModify(c *gin.Context) {
	var (
		request  ModifyOrderRequest
		response Response
	)
	//  validating the request payload via gin framework
	if err := c.BindJSON(&request); err != nil {
		logger.Log.Error("Invalid arguement received", zap.Error(err))
		response.Errors = append(response.Errors, e.ErrorInfo["BadRequest"].GetErrorDetails(""))
		c.JSON(http.StatusBadRequest, response)
		c.Abort()
		return
	}

	//creating rupeeseed api url for bracket order
	uri := rupeeseedObj.EndPoint + BoModifyOrderAPI
	//creating request body for calling rupeeseed api
	requestBody := parseVendorRequestBody(c, request)
	//call rupeeseed modify BoOrderModify api
	body, status, err := s.restCaller.InvokeHttp(http.MethodPost, uri, requestBody, rupeeseedHeaders, ApiTimeout)
	if err != nil {
		logger.Log.Error("BoOrderModify: rupeseed api failure", zap.Error(err), zap.String("api:", uri))
		response.Errors = append(response.Errors, e.ErrorInfo["VendorApiFailure"].GetErrorDetails(""))
		c.JSON(http.StatusInternalServerError, response)
		c.Abort()
		return
	} else if status != http.StatusOK {
		logger.Log.Error("BoOrderModify: rupeseed api failure, failed to modify bracket order", zap.Error(err), zap.String("api:", uri))
		response.Errors = append(response.Errors, e.ErrorInfo["VendorConnectionFailure"].GetErrorDetails(""))
		c.JSON(status, response)
		c.Abort()
		return
	}
	// struct for parsing the rupeeseed api response
	var obj RupeeseedNormalOrderResponse
	// unmarshal rupeeseed response
	err = json.Unmarshal(body, &obj)
	// error occured while unmarshal the response
	if err != nil {
		logger.Log.Error("Failed to marshal the error", zap.Error(err), zap.Any("recevied", string(body)))
		response.Errors = append(response.Errors, e.ErrorInfo["JsonUnmarshalError"].GetErrorDetails(""))
		c.JSON(http.StatusInternalServerError, response)
		c.Abort()
		return
	}
	// handling the failure of bracket order modification at rupeeseed
	if obj.Status != Success {
		logger.Log.Error("bracket order modify api failure", zap.String("msg", obj.Message), zap.String("errorCode", obj.ErrCode))
		if e.RupeeseedErrors[obj.ErrCode] == http.StatusBadRequest {
			response.Errors = append(response.Errors, e.ErrorInfo["BadRequest"].GetErrorDetails(fmt.Sprintf(":%s", obj.Message))) // Once error code recevies , need to handle errors
			c.JSON(http.StatusBadRequest, response)
		} else if e.RupeeseedErrors[obj.ErrCode] == http.StatusInternalServerError {
			response.Errors = append(response.Errors, e.ErrorInfo["InternalServerError"].GetErrorDetails(fmt.Sprintf(":%s", obj.Message)))
			c.JSON(http.StatusInternalServerError, response)
		} else {
			response.Errors = append(response.Errors, e.ErrorInfo["InternalServerError"].GetErrorDetails(""))
			c.JSON(http.StatusInternalServerError, response)
		}
		c.Abort()
		return
	}
	response.Data = obj.Data
	response.Status = true
	response.Message = OrderModificationSuccess
	c.JSON(http.StatusOK, response)
}

/*
handles modification bracket order with product V(CO – Cover Order)
segment E(equity), D(derivative)
with order type MKT(market), LMT(limit) &
validity IOC(Immediate or cance), DAY(Intraday)
*/
func (s *trade) CoverOrderModify(c *gin.Context) {
	var (
		request  ModifyOrderRequest
		response Response
	)
	//  validating the request payload via gin framework
	if err := c.BindJSON(&request); err != nil {
		logger.Log.Error("Invalid arguement received", zap.Error(err))
		response.Errors = append(response.Errors, e.ErrorInfo["BadRequest"].GetErrorDetails(""))
		c.JSON(http.StatusBadRequest, response)
		c.Abort()
		return
	}

	//creating rupeeseed api url for CoOrderModify
	uri := rupeeseedObj.EndPoint + CoModifyOrderApi
	//call rupeeseed modify CoOrderModify api
	requestBody := parseVendorRequestBody(c, request)
	body, status, err := s.restCaller.InvokeHttp(http.MethodPost, uri, requestBody, rupeeseedHeaders, ApiTimeout)
	if err != nil {
		logger.Log.Error("CoverOrderModify: rupeseed api failure", zap.Error(err), zap.String("api:", uri))
		response.Errors = append(response.Errors, e.ErrorInfo["VendorApiFailure"].GetErrorDetails(""))
		c.JSON(http.StatusInternalServerError, response)
		c.Abort()
		return
	} else if status != http.StatusOK {
		logger.Log.Error("CoverOrderModify: rupeseed api failure", zap.Error(err), zap.String("api:", uri))
		response.Errors = append(response.Errors, e.ErrorInfo["VendorConnectionFailure"].GetErrorDetails(""))
		c.JSON(status, response)
		c.Abort()
		return
	}
	// struct for parsing the rupeeseed api response
	var obj RupeeseedNormalOrderResponse
	// unmarshal rupeeseed response
	err = json.Unmarshal(body, &obj)
	// error occured while unmarshal the response
	if err != nil {
		logger.Log.Error("Failed to marshal the error", zap.Error(err), zap.Any("recevied", string(body)))
		response.Errors = append(response.Errors, e.ErrorInfo["JsonUnmarshalError"].GetErrorDetails(""))
		c.JSON(http.StatusInternalServerError, response)
		c.Abort()
		return
	}
	// handling the failure of cover order modification at rupeeseed
	if obj.Status != Success {
		logger.Log.Error("cover order modify api failure", zap.String("msg", obj.Message), zap.String("errorCode", obj.ErrCode))
		if e.RupeeseedErrors[obj.ErrCode] == http.StatusBadRequest {
			response.Errors = append(response.Errors, e.ErrorInfo["BadRequest"].GetErrorDetails(fmt.Sprintf(":%s", obj.Message))) // Once error code recevies , need to handle errors
			c.JSON(http.StatusBadRequest, response)
		} else if e.RupeeseedErrors[obj.ErrCode] == http.StatusInternalServerError {
			response.Errors = append(response.Errors, e.ErrorInfo["InternalServerError"].GetErrorDetails(fmt.Sprintf(":%s", obj.Message)))
			c.JSON(http.StatusInternalServerError, response)
		} else {
			response.Errors = append(response.Errors, e.ErrorInfo["InternalServerError"].GetErrorDetails(""))
			c.JSON(http.StatusInternalServerError, response)
		}
		c.Abort()
		return
	}
	response.Data = obj.Data
	response.Status = true
	response.Message = OrderModificationSuccess
	c.JSON(http.StatusOK, response)
}

/*
api will convert product type of relevant open position

steps within api
  - fetch position book
  - identify open positions
  - match request parameters with open positions (product, position type(B.S), exchange, security id)
  - check if surplus quantity for conversion is available
  - call ruppeeseed api to convert
  - analyse result and reply
*/
func (s *trade) ConvertPosition(c *gin.Context) {
	//fetch position book
	var (
		request  ConvertPositionRequest
		response ConvertPositionResponse
	)
	if err := c.BindJSON(&request); err != nil {
		logger.Log.Error("Invalid arguement received for Convert Position", zap.Error(err))
		response.Errors = append(response.Errors, e.ErrorInfo["BadRequest"].GetErrorDetails(""))
		c.JSON(http.StatusBadRequest, response)
		c.Abort()
		return
	}
	request.UserID = c.GetString("userId")

	//fetch the net position for the user and check if there is any position matching this conversion requirement
	obj, err := s.fetchPositionBook(c)
	if err != nil {
		response.Errors = append(response.Errors, *err)
		c.JSON(http.StatusInternalServerError, response)
		c.Abort()
		return
	}

	found := false
	var position RupeeSeedPositionBook
	for _, position = range obj.Data {
		qty := position.NetQty
		//do not check for closed positions
		if qty == 0 {
			continue
		}
		var txType string
		if qty > 0 {
			txType = BUY
		} else {
			txType = SELL
			//convert net quqntity to positive value for comparision ahead
			qty = qty * -1
		}
		secId := position.SecurityID
		exchg := position.Exchange
		product := position.Product

		//check for open position with same exchange, security id, product and surplus quantity for conversion
		if txType == request.PositionType &&
			exchg == request.Exchange &&
			secId == request.ExchangeToken &&
			product == request.PositionFrom &&
			qty >= request.Quantity {
			request.Segment = position.Segment
			found = true
			break
		} else {
			request.Segment = "E"
		}
	}
	if !found {
		logger.Log.Info("No open positions available to convert")
		response.Errors = append(response.Errors, e.ErrorInfo["NoDataFound"].GetErrorDetails("no matching open positions available to convert"))
		c.JSON(http.StatusBadRequest, response)
		c.Abort()
		return
	}

	//call ruppeeseed api for position conversion
	convertPosReq := getRupeeseedConvertPositionRequestBody(c, request)
	obj1, err := s.convertPosition(convertPosReq)
	if err != nil {
		if err.ErrName == e.VendorOMSError {
			response.Status = false
			response.Data = obj1.Message
			c.JSON(http.StatusOK, response)
			return
		}
		response.Errors = append(response.Errors, *err)
		c.JSON(http.StatusInternalServerError, response)
		c.Abort()
		return
	}
	response.Data = obj1.Message
	response.Status = true
	c.JSON(http.StatusOK, response)
}

/*
fetch Position Book from Ruppeeseed
any error from invoking Ruppeeseed API is
converted to Error struct from ("equity-trading/pkg/errors") package

	input:
		context
	output:
		RuppeeseedPositionBookResponse
		*Error
*/
func (s *trade) fetchPositionBook(c *gin.Context) (RupeeseedPositionBookResponse, *e.Error) {
	var (
		obj RupeeseedPositionBookResponse
		er  e.Error
	)

	st := time.Now()
	//creating rupeeseed api url for NetPosition
	uri := rupeeseedObj.EndPoint + PositionBookApi
	//creating request body for calling rupeeseed NetPosition api
	positionRequestBody := getPositionBookRupeeseedRequestBody(c)
	//call rupeeseed NetPosition api
	body, status, err := s.restCaller.InvokeResty(http.MethodPost, uri, positionRequestBody, nil, 700)
	logger.Log.Info("api details", zap.Any("latency", time.Since(st)), zap.Any("status", status), zap.Error(err), zap.Any("data", string(body)))

	if err != nil {
		logger.Log.Error("PositionBook rupeeseed api failure", zap.Error(err), zap.String("api:", uri))
		er = e.ErrorInfo["VendorApiFailure"].GetErrorDetails("")
		return obj, &er
	} else if status != http.StatusOK {
		// Rupeeseed api unable to retreive PositionBook
		logger.Log.Error("NetPosition: rupeeseed api failure, failed to retreive positionBook", zap.Error(err), zap.String("api:", uri))
		er = e.ErrorInfo["VendorConnectionFailure"].GetErrorDetails("")
		return obj, &er
	}
	// unmarshal rupeeseed response
	err = json.Unmarshal(body, &obj)
	if err != nil {
		logger.Log.Error("Failed to marshal the error", zap.Error(err), zap.Any("recevied", string(body)))
		er = e.ErrorInfo["JsonUnmarshalError"].GetErrorDetails("")
		return obj, &er
	}
	// handling the failure of PositionBook at rupeeseed
	if obj.Status != Success {
		logger.Log.Error("NetPosition api failure", zap.String("msg", obj.Message), zap.String("errorCode", obj.ErrorCode))
		if e.RupeeseedErrors[obj.ErrorCode] == http.StatusBadRequest {
			er = e.ErrorInfo["BadRequest"].GetErrorDetails(fmt.Sprintf(":%s", obj.Message))
			return obj, &er
		} else if e.RupeeseedErrors[obj.ErrorCode] == http.StatusInternalServerError {
			er = e.ErrorInfo["InternalServerError"].GetErrorDetails(fmt.Sprintf(":%s", obj.Message))
			return obj, &er
		} else {
			er = e.ErrorInfo["InternalServerError"].GetErrorDetails("")
			return obj, &er
		}
	}
	return obj, nil
}

/*
call Ruppeeseed API to convert product type for open positions
any error from from Ruppeeseed API is handled based on type of
error and returned as custom error ("equity-trading/pkg/errors")

	input:
		RuppeeseedConvertPositionRequest
	output:
		RuppeeseedConvertPositionResponse
		*Error
*/
func (s *trade) convertPosition(request RuppeeseedConvertPositionRequest) (RuppeeseedConvertPositionResponse, *e.Error) {
	var (
		obj RuppeeseedConvertPositionResponse
		er  e.Error
	)

	st := time.Now()
	uri := rupeeseedObj.EndPoint + ConvertPositionApi
	logger.Log.Info("created convert position request", zap.Any("request", request))
	body, status, err := s.restCaller.InvokeResty(http.MethodPost, uri, request, nil, 1000)
	logger.Log.Info("api details", zap.Any("lateny", time.Since(st)), zap.Any("status", status), zap.Error(err), zap.Any("data", string(body)))
	if err != nil || status != http.StatusOK {
		logger.Log.Error("convert position rupeeseed api failure", zap.Error(err), zap.String("api:", uri))
		er = e.ErrorInfo["VendorApiFailure"].GetErrorDetails("")
		return obj, &er
	}

	err = json.Unmarshal(body, &obj)
	if err != nil {
		logger.Log.Error("Failed to marshal the error", zap.Error(err), zap.Any("recevied", string(body)))
		er = e.ErrorInfo["JsonUnmarshalError"].GetErrorDetails("")
		return obj, &er
	}
	if obj.Status != Success {
		logger.Log.Error("order entry api failure", zap.String("msg", obj.Message), zap.String("errorCode", obj.ErrorCode))
		if obj.ErrorCode == "RS-0022" {
			er = e.ErrorInfo["VendorOMSError"].GetErrorDetails("")
			return obj, &er
		} else {
			er = e.ErrorInfo["VendorApiFailure"].GetErrorDetails(fmt.Sprintf(":%s", obj.Message)) // Once error code recevied , need to handle errors
			return obj, &er
		}
	}
	return obj, nil

}

/*
validating business constraints for placing
cover order
*/
func coverOrderValidation(request PlaceCoverOrderRequest) error {
	//limit trigger order validation

	if request.OrderType == LMT && request.Price <= 0.0 {
		return errors.New(":Price cannot be zero with limit order")
	}

	if request.TriggerPrice <= 0.0 {
		return errors.New(":Trigger Price cannot be zero")
	}

	if request.OffMktFlag && request.OffMktOrderTimeFlag <= 0 {
		return errors.New(":OffMktOrderTimeFlag possible allowed values are 1|2|3 in AMO")
	}

	return nil
}

/*
validating business constraints for placing
bracket order
*/
func bracketOrderValidation(request PlaceBracketOrderRequest) error {
	//limit trigger order validation

	if request.OrderType == LMT && request.Price <= 0.0 {
		return errors.New(":Price cannot be zero with limit order")
	}

	if request.ProfitValue <= 0.0 && request.StoplossValue <= 0.0 {
		return errors.New(":ProfitValue, StoplossValue cannot be zero")
	}

	if request.ProfitValue <= 0.0 {
		return errors.New(":ProfitValue cannot be zero")
	}
	if request.StoplossValue <= 0.0 {
		return errors.New(":StoplossValue cannot be zero")
	}

	if request.OffMktFlag && request.OffMktOrderTimeFlag <= 0 {
		return errors.New(":OffMktOrderTimeFlag possible allowed values are 1|2|3 in AMO")
	}

	return nil
}
