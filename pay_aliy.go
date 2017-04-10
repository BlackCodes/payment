package payment

import (
	"encoding/json"
	"fmt"

	"facework.im/share/logging"
	//"strings"
	"strconv"
)

type AliyPay struct {
	Payment
}

const (
	APAY_URL     = "https://mapi.alipay.com/gateway.do"
	SERVCIE_NAME = "create_direct_pay_by_user"
	PARTNER      = "partner"
	KEY          = "key"
	RETURN_URL   = "return_url"
	NOTIFY_URL   = "notify_url"
	SELLER       = "seller"
)

// 订单单数据包
type OrderJSON struct {
	ExtraParam  string  `json:"extra_common_param"`
	Service     string  `json:"service"`
	Partner     string  `json:"partner"`
	InputChar   string  `json:"_input_charset"`
	SignType    string  `json:"sign_type"`
	Sign        string  `json:"sign"`
	NotifyUrl   string  `json:"notify_url"`
	ReturnUrl   string  `json:"return_url"`
	OrderID     string  `json:"out_trade_no"`
	Subject     string  `json:"subject"` // 商品名称
	PaymentType int     `json:"payment_type"`
	TotalFee    float32 `json:"total_fee"`
	SellerID    string  `json:"seller_id"` // 卖家支付宝用户号
	Body        string  `json:"body"`      // 商品描述
	ClientIP    string  `json:"exter_invoke_ip"`
}

func NewAlipay(params map[string]interface{}) *AliyPay {
	return &AliyPay{Payment{Config: params}}
}

func (this *AliyPay) GenderPayUrl(order Order) (string, error) {
	var o OrderJSON
	o.ExtraParam = order.ExtraParam
	o.Service = SERVCIE_NAME
	o.Partner = this.getConfig(PARTNER).(string)
	o.InputChar = "utf-8"
	o.NotifyUrl = this.getConfig(NOTIFY_URL).(string)
	o.ReturnUrl = this.getConfig(RETURN_URL).(string)
	o.Subject = order.ProudctName
	o.Body = order.ProudctDescription
	o.OrderID = order.OrderID
	o.ClientIP = order.IP
	o.PaymentType = 1

	if sellerId := this.getConfig(SELLER); sellerId == nil {
		o.SellerID = this.getConfig(PARTNER).(string)
	} else {
		o.SellerID = sellerId.(string)
	}
	o.TotalFee = float32(order.PriceTotal) / 100
	params := this.struct2map(o)

	o.Sign = this.makeSign(params)
	o.SignType = "MD5"
	urlPrams := this.struct2map(o)
	return fmt.Sprintf("%s?%s", APAY_URL, this.makeUrl(urlPrams)), nil

}
func (this *AliyPay) PayNotify(notify []byte) (*NofiyData, error) {

	if len(notify) == 0 {
		return nil, fmt.Errorf("Notify Data empty")
	}
	var resultData NofiyData
	var notifyData map[string]interface{}
	notifyMap := make(map[string]interface{}, len(notifyData))
	json.Unmarshal(notify, &notifyData)
	for k, v := range notifyData {
		if k == "sign_type" {
			continue
		}
		if item, ok := v.([]interface{}); ok {
			notifyMap[k] = item[0]
		}

	}
	if notifyMap["trade_status"].(string) != "TRADE_SUCCESS" {
		resultData.ReturnData = []byte("Status Error")
		return &resultData, fmt.Errorf("Status Error")
	}
	if !this.checkSign(notifyMap) {

		resultData.ReturnData = []byte("Sign Error")
		return &resultData, fmt.Errorf("Sign Error")
	}
	resultData.OrderID = notifyMap["out_trade_no"].(string)
	tfee, _ := strconv.ParseFloat(notifyMap["total_fee"].(string), 32)
	resultData.TotalFee = int(tfee * 100)
	resultData.TransactionID = notifyMap["trade_no"].(string)
	resultData.ReturnData = []byte("success")
	return &resultData, nil
}

func (this *AliyPay) getConfig(name string) interface{} {
	if v, ok := this.Config[name]; !ok {
		return nil
	} else {
		return v
	}
}

func (this *AliyPay) checkConfig() bool {
	configFiled := []string{PARTNER, SELLER, KEY, NOTIFY_URL, RETURN_URL}
	for _, item := range configFiled {
		if _, ok := this.Payment.Config[item]; !ok {
			logging.Error("Not found [%v] in the config map", item)
			return false
		}
	}
	return true
}

func (this *AliyPay) makeSign(params map[string]interface{}) string {
	signParams := this.makeUrl(params)
	return this.MD5Sigin(fmt.Sprintf("%s%s", signParams, this.getConfig(KEY)))
}

func (this *AliyPay) checkSign(params map[string]interface{}) bool {
	originSign := params["sign"]
	delete(params, "sign")
	paySign := this.makeSign(params)
	if originSign == paySign {
		return true
	} else {
		return false
	}
}
