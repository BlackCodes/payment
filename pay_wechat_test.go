package payment

import (
	"fmt"
	"testing"
)

func Test_makeUrl(t *testing.T) {
	//{"mch_id", "appid", "key", "notify_url"}
	wxconfig := map[string]interface{}{
		"appid":      "",
		"mch_id":     "",
		"key":        "",
		"notify_url": "",
	}
	pay, err := NewPayment().Init("wx", wxconfig)
	if err != nil {
		fmt.Println("Happend err", err)
		return
	}

	oreder := Order{OrderID: "123212323222", ProudctName: "I am test", PriceTotal: 1, ProductID: 10001, IP: "127.0.0.1"}
	url, err := pay.GenderPayUrl(oreder)
	fmt.Println(url, err)
}
