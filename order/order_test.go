package trade

import (
	"bytes"
	"encoding/json"
	config "equity-trading/pkg/config"
	db "equity-trading/pkg/db"
	dbmock "equity-trading/pkg/db/mock"
	"equity-trading/pkg/logger"
	"equity-trading/pkg/utils"
	mock "equity-trading/pkg/utils/mock"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"go.uber.org/zap"
)

type OrderData struct {
	Order string `json:"order_no"`
}

func getConntext(method string, data interface{}) *gin.Context {
	recorder := httptest.NewRecorder()
	temp, _ := gin.CreateTestContext(recorder)
	temp.Request = &http.Request{Method: method}
	if method == "POST" || method == "PUT" {
		byteData, err := json.Marshal(data)
		if err != nil {
			logger.Log.Error("failed to generate context", zap.Error(err))
			return nil
		}
		temp.Request.Header = http.Header{}
		temp.Request.Header.Set("Content-Type", "application/json")
		temp.Request.Body = io.NopCloser(bytes.NewBuffer(byteData))
	} else {

		//temp.AddParam("id", "1")  // add key-value pair for get request
	}
	return temp
}

func TestPlaceOrder(t *testing.T) {
	var (
		dbObj       db.DBLayer
		restCaller  utils.RestCaller
		redisCaller utils.RedisInterface
	)

	tests := []struct {
		name       string
		input      PlaceOrderRequest
		setup      func(*gin.Context, PlaceOrderRequest)
		output     PlaceOrderResponse
		httpStatus bool
		wantErr    bool
		ctx        *gin.Context
	}{
		{
			name: "InvalidTxnType",
			input: PlaceOrderRequest{
				TxnType:       "D",
				Exchange:      "NSE",
				Segment:       "E",
				Product:       "C",
				ExchangeToken: 1594,
				Quantity:      1,
				Validity:      "DAY",
				OrderType:     "MKT",
			},
			setup: func(c *gin.Context, data PlaceOrderRequest) {
				ctrl := gomock.NewController(t)
				invoker := mock.NewMockUtils(ctrl)
				repo := dbmock.NewMockDBLayer(ctrl)
				dbObj = repo
				redisCaller = invoker
				restCaller = invoker
			},
			output:  PlaceOrderResponse{},
			wantErr: true,
		},
		{
			name: "InvalidSegmentType",
			input: PlaceOrderRequest{
				TxnType:       "B",
				Exchange:      "NSE",
				Segment:       "A",
				Product:       "C",
				ExchangeToken: 1594,
				Quantity:      1,
				Validity:      "DAY",
				OrderType:     "MKT",
			},
			setup: func(c *gin.Context, data PlaceOrderRequest) {
				ctrl := gomock.NewController(t)
				invoker := mock.NewMockUtils(ctrl)
				repo := dbmock.NewMockDBLayer(ctrl)
				dbObj = repo
				redisCaller = invoker
				restCaller = invoker
			},
			output:  PlaceOrderResponse{},
			wantErr: true,
		},
		{
			name: "InvalidOrderType",
			input: PlaceOrderRequest{
				TxnType:       "B",
				Exchange:      "NSE",
				Segment:       "A",
				Product:       "C",
				ExchangeToken: 1594,
				Quantity:      1,
				Validity:      "DAY",
				OrderType:     "MLLKT",
			},
			setup: func(c *gin.Context, data PlaceOrderRequest) {
				ctrl := gomock.NewController(t)
				invoker := mock.NewMockUtils(ctrl)
				repo := dbmock.NewMockDBLayer(ctrl)
				dbObj = repo
				redisCaller = invoker
				restCaller = invoker
			},
			output:  PlaceOrderResponse{},
			wantErr: true,
		},
		{
			name: "InvalidValidity",
			input: PlaceOrderRequest{
				TxnType:       "B",
				Exchange:      "NSE",
				Segment:       "A",
				Product:       "C",
				ExchangeToken: 1594,
				Quantity:      1,
				Validity:      "DA1Y",
				OrderType:     "MKT",
			},
			setup: func(c *gin.Context, data PlaceOrderRequest) {
				ctrl := gomock.NewController(t)
				invoker := mock.NewMockUtils(ctrl)
				repo := dbmock.NewMockDBLayer(ctrl)
				dbObj = repo
				redisCaller = invoker
				restCaller = invoker
			},
			output:  PlaceOrderResponse{},
			wantErr: true,
		},
		{
			name: "InvalidProductType",
			input: PlaceOrderRequest{
				TxnType:       "B",
				Exchange:      "NSE",
				Segment:       "E",
				Product:       "X",
				ExchangeToken: 1594,
				Quantity:      1,
				Validity:      "DAY",
				OrderType:     "MKT",
			},
			setup: func(c *gin.Context, data PlaceOrderRequest) {
				ctrl := gomock.NewController(t)
				invoker := mock.NewMockUtils(ctrl)
				repo := dbmock.NewMockDBLayer(ctrl)
				dbObj = repo
				redisCaller = invoker
				restCaller = invoker
			},
			output:  PlaceOrderResponse{},
			wantErr: true,
		},
		{
			name: "InvalidQuantity",
			input: PlaceOrderRequest{
				TxnType:       "B",
				Exchange:      "NSE",
				Segment:       "E",
				Product:       "C",
				ExchangeToken: 0,
				Quantity:      -1,
				Validity:      "DAY",
				OrderType:     "MKT",
			},
			setup: func(c *gin.Context, data PlaceOrderRequest) {
				ctrl := gomock.NewController(t)
				invoker := mock.NewMockUtils(ctrl)
				repo := dbmock.NewMockDBLayer(ctrl)
				dbObj = repo
				redisCaller = invoker
				restCaller = invoker
			},
			output:  PlaceOrderResponse{},
			wantErr: true,
		},
		{
			name: "TriggerOrderInvalidLMTPrice",
			input: PlaceOrderRequest{
				TxnType:       "B",
				Exchange:      "NSE",
				Segment:       "E",
				Product:       "C",
				ExchangeToken: 0,
				Quantity:      -1,
				Validity:      "DAY",
				OrderType:     "SL",
				Price:         0.0,
				TriggerPrice:  0.0,
			},
			setup: func(c *gin.Context, data PlaceOrderRequest) {
				ctrl := gomock.NewController(t)
				invoker := mock.NewMockUtils(ctrl)
				repo := dbmock.NewMockDBLayer(ctrl)
				dbObj = repo
				redisCaller = invoker
				restCaller = invoker
			},
			output:  PlaceOrderResponse{},
			wantErr: true,
		},
		{
			name: "TriggerOrderInvalidLMTPriceBuy",
			input: PlaceOrderRequest{
				TxnType:       "B",
				Exchange:      "NSE",
				Segment:       "E",
				Product:       "C",
				ExchangeToken: 0,
				Quantity:      -1,
				Validity:      "DAY",
				OrderType:     "SL",
				Price:         10.0,
				TriggerPrice:  100.0,
			},
			setup: func(c *gin.Context, data PlaceOrderRequest) {
				ctrl := gomock.NewController(t)
				invoker := mock.NewMockUtils(ctrl)
				repo := dbmock.NewMockDBLayer(ctrl)
				dbObj = repo
				redisCaller = invoker
				restCaller = invoker
			},
			output:  PlaceOrderResponse{},
			wantErr: true,
		},
		{
			name: "TriggerOrderInvalidLMTPriceSell",
			input: PlaceOrderRequest{
				TxnType:       "S",
				Exchange:      "NSE",
				Segment:       "E",
				Product:       "C",
				ExchangeToken: 0,
				Quantity:      -1,
				Validity:      "DAY",
				OrderType:     "SL",
				Price:         100.0,
				TriggerPrice:  10.0,
			},
			setup: func(c *gin.Context, data PlaceOrderRequest) {
				ctrl := gomock.NewController(t)
				invoker := mock.NewMockUtils(ctrl)
				repo := dbmock.NewMockDBLayer(ctrl)
				dbObj = repo
				redisCaller = invoker
				restCaller = invoker
			},
			output:  PlaceOrderResponse{},
			wantErr: true,
		},
		{
			name: "InvalidExchangetoken",
			input: PlaceOrderRequest{
				TxnType:       "B",
				Exchange:      "NSE",
				Segment:       "E",
				Product:       "C",
				ExchangeToken: 0,
				Quantity:      1,
				Validity:      "DAY",
				OrderType:     "MKT",
			},
			setup: func(c *gin.Context, data PlaceOrderRequest) {
				ctrl := gomock.NewController(t)
				invoker := mock.NewMockUtils(ctrl)
				repo := dbmock.NewMockDBLayer(ctrl)
				dbObj = repo
				redisCaller = invoker
				restCaller = invoker
			},
			output:  PlaceOrderResponse{},
			wantErr: true,
		},
		{
			name: "Orderplaced",
			input: PlaceOrderRequest{
				TxnType:       "B",
				Exchange:      "NSE",
				Segment:       "E",
				Product:       "C",
				ExchangeToken: 1594,
				Quantity:      1,
				Validity:      "DAY",
				OrderType:     "MKT",
			},
			setup: func(c *gin.Context, data PlaceOrderRequest) {
				ctrl := gomock.NewController(t)
				invoker := mock.NewMockUtils(ctrl)
				repo := dbmock.NewMockDBLayer(ctrl)
				dbObj = repo
				redisCaller = invoker
				restCaller = invoker
				req := getRupeeseedOrderRequestBody(c, data)
				resp := RupeeseedNormalOrderResponse{
					Status:  "success",
					ErrCode: "RS-0022",
					Message: "Order submitted successfully. Your Order Ref No. 112211242008",
					Data: []OrderData{
						{
							Order: "112211242008",
						},
					},
					IV: "",
				}
				byteData, _ := json.Marshal(resp)
				uri := config.GetConfig().GetString("rupeeseed.endpoint") + OrderApi
				invoker.EXPECT().InvokeHttp(http.MethodPost, uri, req, rupeeseedHeaders, ApiTimeout).Return(byteData, http.StatusOK, nil).Times(1)
			},
			output:  PlaceOrderResponse{},
			wantErr: false,
		},
		{
			name: "VendorApifailure-InternalServerError",
			input: PlaceOrderRequest{
				TxnType:       "B",
				Exchange:      "NSE",
				Segment:       "E",
				Product:       "C",
				ExchangeToken: 1594,
				Quantity:      1,
				Validity:      "DAY",
				OrderType:     "MKT",
			},
			setup: func(c *gin.Context, data PlaceOrderRequest) {
				ctrl := gomock.NewController(t)
				invoker := mock.NewMockUtils(ctrl)
				repo := dbmock.NewMockDBLayer(ctrl)
				dbObj = repo
				redisCaller = invoker
				restCaller = invoker
				req := getRupeeseedOrderRequestBody(c, data)
				resp := RupeeseedNormalOrderResponse{
					Status:  "error",
					ErrCode: "RS-0023",
					Message: "SCRIP IS BLOCKED",
					Data: []OrderData{
						{
							Order: "112211242008",
						},
					},
					IV: "",
				}
				byteData, _ := json.Marshal(resp)
				uri := config.GetConfig().GetString("rupeeseed.endpoint") + OrderApi
				invoker.EXPECT().InvokeHttp(http.MethodPost, uri, req, rupeeseedHeaders, ApiTimeout).Return(byteData, http.StatusInternalServerError, nil).Times(1)
			},
			output:  PlaceOrderResponse{},
			wantErr: true,
		},
		{
			name: "VendorapiFailure-BadRequest",
			input: PlaceOrderRequest{
				TxnType:       "B",
				Exchange:      "NSE",
				Segment:       "E",
				Product:       "C",
				ExchangeToken: 1594,
				Quantity:      1,
				Validity:      "DAY",
				OrderType:     "MKT",
			},
			setup: func(c *gin.Context, data PlaceOrderRequest) {
				ctrl := gomock.NewController(t)
				invoker := mock.NewMockUtils(ctrl)
				repo := dbmock.NewMockDBLayer(ctrl)
				dbObj = repo
				redisCaller = invoker
				restCaller = invoker
				req := getRupeeseedOrderRequestBody(c, data)
				resp := RupeeseedNormalOrderResponse{
					Status:  "error",
					ErrCode: "RS-0023",
					Message: "SCRIP IS BLOCKED",
					Data: []OrderData{
						{
							Order: "112211242008",
						},
					},
					IV: "",
				}
				byteData, _ := json.Marshal(resp)
				uri := config.GetConfig().GetString("rupeeseed.endpoint") + OrderApi
				invoker.EXPECT().InvokeHttp(http.MethodPost, uri, req, rupeeseedHeaders, ApiTimeout).Return(byteData, http.StatusBadRequest, nil).Times(1)
			},
			output:  PlaceOrderResponse{},
			wantErr: true,
		},
		{
			name: "VendorCouldn'tConnect",
			input: PlaceOrderRequest{
				TxnType:       "B",
				Exchange:      "NSE",
				Segment:       "E",
				Product:       "C",
				ExchangeToken: 1594,
				Quantity:      1,
				Validity:      "DAY",
				OrderType:     "MKT",
			},
			setup: func(c *gin.Context, data PlaceOrderRequest) {
				ctrl := gomock.NewController(t)
				invoker := mock.NewMockUtils(ctrl)
				repo := dbmock.NewMockDBLayer(ctrl)
				dbObj = repo
				redisCaller = invoker
				restCaller = invoker
				req := getRupeeseedOrderRequestBody(c, data)
				resp := RupeeseedNormalOrderResponse{
					Status:  "error",
					ErrCode: "RS-0023",
					Message: "SCRIP IS BLOCKED",
					Data: []OrderData{
						{
							Order: "112211242008",
						},
					},
					IV: "",
				}
				byteData, _ := json.Marshal(resp)
				uri := config.GetConfig().GetString("rupeeseed.endpoint") + OrderApi
				invoker.EXPECT().InvokeHttp(http.MethodPost, uri, req, rupeeseedHeaders, ApiTimeout).Return(byteData, http.StatusNotFound, nil).Times(1)
			},
			output:  PlaceOrderResponse{},
			wantErr: true,
		},
		{
			name: "ApiCallingFailure",
			input: PlaceOrderRequest{
				TxnType:       "B",
				Exchange:      "NSE",
				Segment:       "E",
				Product:       "C",
				ExchangeToken: 1594,
				Quantity:      1,
				Validity:      "DAY",
				OrderType:     "MKT",
			},
			setup: func(c *gin.Context, data PlaceOrderRequest) {
				ctrl := gomock.NewController(t)
				invoker := mock.NewMockUtils(ctrl)
				repo := dbmock.NewMockDBLayer(ctrl)
				dbObj = repo
				redisCaller = invoker
				restCaller = invoker
				req := getRupeeseedOrderRequestBody(c, data)
				uri := config.GetConfig().GetString("rupeeseed.endpoint") + OrderApi
				invoker.EXPECT().InvokeHttp(http.MethodPost, uri, req, rupeeseedHeaders, ApiTimeout).Return(nil, 0, errors.New("ApiFormatError")).Times(1)
			},
			output:  PlaceOrderResponse{},
			wantErr: true,
		},
		{
			name: "InvalidResponseReceiveFromVendor",
			input: PlaceOrderRequest{
				TxnType:       "B",
				Exchange:      "NSE",
				Segment:       "E",
				Product:       "C",
				ExchangeToken: 1594,
				Quantity:      1,
				Validity:      "DAY",
				OrderType:     "MKT",
			},
			setup: func(c *gin.Context, data PlaceOrderRequest) {
				ctrl := gomock.NewController(t)
				invoker := mock.NewMockUtils(ctrl)
				repo := dbmock.NewMockDBLayer(ctrl)
				dbObj = repo
				redisCaller = invoker
				restCaller = invoker
				req := getRupeeseedOrderRequestBody(c, data)
				uri := config.GetConfig().GetString("rupeeseed.endpoint") + OrderApi
				invoker.EXPECT().InvokeHttp(http.MethodPost, uri, req, rupeeseedHeaders, ApiTimeout).Return([]byte("SORRY"), 0, errors.New("ApiFormatError")).Times(1)
			},
			output:  PlaceOrderResponse{},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.ctx = getConntext("POST", test.input)
			test.setup(test.ctx, test.input)
			servObj := NewTradeGroup(dbObj, restCaller, redisCaller)
			servObj.PlaceOrder(test.ctx)
			if test.wantErr {
				if !test.ctx.IsAborted() {
					t.Errorf("TestPlaceOrder() failed testcase=[%s] want contextaborted, got  [%v]", test.name, test.ctx.IsAborted())
					return
				} else {
					// Need to validate the output response R&D
				}
			}
			fmt.Println("Test case passed :", test.name)
		})
	}

}
