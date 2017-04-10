package payment

import (
	"bytes"
	"encoding/xml"
	"facework.im/share/logging"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type Wechat struct {
	Payment
}

const (
	PAY_URL     = "https://api.mch.weixin.qq.com/pay/unifiedorder"
	MERCHANT_ID = "mch_id"
	APP_ID      = "appid"
	APP_KEY     = "key"
)

// 统一下单数据包
type OrderXML struct {
	XMLName    xml.Name `xml:"xml"`
	APPID      string   `xml:"appid"`
	MchID      string   `xml:"mch_id"`
	Body       string   `xml:"body"`
	Detail     string   `xml:"detail"`
	NonceStr   string   `xml:"nonce_str"`
	NotifyUrl  string   `xml:"notify_url"`
	OrderID    string   `xml:"out_trade_no"`
	ClientIP   string   `xml:"spbill_create_ip"`
	PriceTotal int      `xml:"total_fee"`
	TradeType  string   `xml:"trade_type"`
	ProductId  int      `xml:"product_id"`
	Sign       string   `xml:"sign"`
}

// 微信统一下单返回数据包
type WeChatPayXML struct {
	ReturnCode string `xml:"return_code"`
	ReturnMsg  string `xml:"return_msg"`
	MchID      string `xml:"mch_id"`
	APPID      string `xml:"appid"`
	NonceStr   string `xml:"nonce_str"`
	Sign       string `xml:"sign"`
	ResultCode string `xml:"result_code"`
	PrepayID   string `xml:"prepay_id"`
	TradeType  string `xml:"trade_type"`
	CodeUrl    string `xml:"code_url"`
	ErrorMSG   string `xml:"err_code_des"`
}

// 微信异步通知数据包
type WechatNotify struct {
	APPID         string `xml:"appid"`
	BankType      string `xml:"bank_type"`
	CashFee       string `xml:"cash_fee"` // 现金支付金额
	FeeType       string `xml:"fee_type"` // 货币类型
	IsSubscribe   string `xml:"is_subscribe"`
	MchID         string `xml:"mch_id"`
	NonceStr      string `xml:"nonce_str"`
	OpenID        string `xml:"openid"`
	OrderID       string `xml:"out_trade_no"`
	ResultCode    string `xml:"result_code"`
	ReturnCode    string `xml:"return_code"`
	Sign          string `xml:"sign"`
	TimeEnd       string `xml:"time_end"`
	TotalFee      int    `xml:"total_fee"`
	TradeType     string `xml:"trade_type"`
	TranscationID string `xml:"transaction_id"` // 微信支付订单号

}

type returnMSG struct {
	XMLName    xml.Name `xml:"xml"`
	ReturnCode string   `xml:"return_code"`
	ReturnMSG  string   `xml:"return_msg"`
}

func NewWechat(params map[string]interface{}) *Wechat {
	return &Wechat{Payment{Config: params}}
}

func (this *Wechat) GenderPayUrl(order Order) (string, error) {
	var o OrderXML
	o.APPID = this.getConfig(APP_ID).(string)
	o.MchID = this.getConfig(MERCHANT_ID).(string)
	o.Body = order.ProudctName
	o.Detail = order.ProudctDescription
	o.NonceStr = this.GenerateString(16)
	o.NotifyUrl = this.getConfig(NOTIFY_URL).(string)
	o.OrderID = order.OrderID
	o.ClientIP = order.IP
	o.PriceTotal = order.PriceTotal
	o.TradeType = "NATIVE"
	o.ProductId = order.ProductID
	params := this.struct2map(o)

	o.Sign = this.makeSign(params)
	data, err := xml.Marshal(o)

	r, err := this.httpPostXML(PAY_URL, data)
	if err != nil {
		return "", fmt.Errorf("response Error | %v", err)
	}

	res, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("get body Error | %v", err)
	}
	var wechatPay WeChatPayXML
	xml.Unmarshal(res, &wechatPay)
	if wechatPay.ResultCode == "FAIL" {
		return "", fmt.Errorf("%v", wechatPay.ErrorMSG)
	}
	payMap := this.struct2map(wechatPay)
	if !this.checkSign(payMap) {
		return "", fmt.Errorf("Sign Error")

	} else {
		return wechatPay.CodeUrl, nil
	}

}
func (this *Wechat) PayNotify(notify []byte) (*NofiyData, error) {
	if len(notify) == 0 {
		return nil, fmt.Errorf("Notify Data empty")
	}
	var notifyPay WechatNotify
	var msg returnMSG
	var paydata NofiyData
	xml.Unmarshal(notify, &notifyPay)
	notifyMap := this.struct2map(notifyPay)
	if !this.checkSign(notifyMap) {
		msg.ReturnCode = "FAIL"
		msg.ReturnMSG = "Sign Error"

		paydata.ReturnData, _ = xml.Marshal(msg)
		return &paydata, fmt.Errorf("Sign Error")
	}
	paydata.OrderID = notifyPay.OrderID
	paydata.TotalFee = notifyPay.TotalFee
	paydata.TransactionID = notifyPay.TranscationID
	msg.ReturnMSG = "OK"
	msg.ReturnCode = "SUCCESS"
	paydata.ReturnData, _ = xml.Marshal(msg)
	return &paydata, nil
}

func (this *Wechat) httpPostXML(url string, data []byte) (*http.Response, error) {
	client := &http.Client{}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	resp, err := client.Do(req)
	defer req.Body.Close()
	return resp, err

}

func (this *Wechat) getConfig(name string) interface{} {
	if v, ok := this.Config[name]; !ok {
		return nil
	} else {
		return v
	}
}

func (this *Wechat) checkConfig() bool {
	configFiled := []string{"mch_id", "appid", "key", "notify_url"}
	for _, item := range configFiled {
		if _, ok := this.Payment.Config[item]; !ok {
			logging.Error("Not found [%v] in the config map", item)
			return false
		}
	}
	return true
}

func (this *Wechat) makeSign(params map[string]interface{}) string {
	signParams := this.makeUrl(params)
	return strings.ToUpper(this.MD5Sigin(fmt.Sprintf("%s&key=%s", signParams, this.getConfig(APP_KEY))))
}

func (this *Wechat) checkSign(params map[string]interface{}) bool {
	originSign := params["sign"]
	delete(params, "sign")
	paySign := this.makeSign(params)
	if originSign == paySign {
		return true
	} else {
		return false
	}
}
