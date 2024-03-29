package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/adonese/noebs/ebs_fields"
)

const (
	TMK = "E6FBFD2C914A155D" // TMK Terminal master key
	TWK = "2277898cef81413e" // Terminal working key
)

func main() {
	http.HandleFunc("/workingKey", WorkingKey)
	http.HandleFunc("/purchase", Purchase)
	http.ListenAndServe(":8090", nil)
}

var tranFee float32 = 1.5

func generateError(f ebs_fields.PurchaseFields, status, message string, code int) ebs_fields.GenericEBSResponseFields {
	return ebs_fields.GenericEBSResponseFields{
		ResponseStatus:         status,
		ResponseMessage:        message,
		ResponseCode:           code,
		TerminalID:             f.TerminalID,
		ClientID:               f.ClientID,
		SystemTraceAuditNumber: generateInt(),
		TranAmount:             f.TranAmount,
		TranDateTime:           f.TranDateTime,
		PAN:                    getLastPan(f.Pan),
		TranCurrency:           "USD",
		TranFee:                &tranFee,
	}
}

func getLastPan(pan string) string {
	if len(pan) >= 16 {
		return pan[len(pan)-4:]
	}
	return pan
}

func generateInt() int {
	return rand.Intn(9999)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// WorkingKey static working key for visa purposes.
func WorkingKey(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	successfull := map[string]ebs_fields.GenericEBSResponseFields{
		"ebs_response": {
			ResponseMessage: "Approval",
			ResponseStatus:  "Successful",
			ResponseCode:    0,
			WorkingKey:      "2277898cef81413e",
		},
	}
	w.Write(toJSON(successfull))
}

//Purchase main interface to interact with Enaya's /charge/ api.
func Purchase(w http.ResponseWriter, r *http.Request) {
	var fields ebs_fields.PurchaseFields
	w.Header().Add("content-type", "application/json")

	req, err := ioutil.ReadAll(r.Body)
	if err != nil {

		verr := ebs_fields.ErrorDetails{Message: "EBS Error", Code: 600, Details: generateError(fields, "Failed", err.Error(), 600)}
		log.Printf("The response is: %v", string(toJSON(verr)))
		w.WriteHeader(http.StatusBadGateway)
		w.Write(toJSON(verr))

		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(req, &fields); err != nil {
		log.Printf("Error in unmarshaling request: Error: %v", err)

		verr := ebs_fields.ErrorDetails{Message: "EBS Error", Code: 600, Details: generateError(fields, "Failed", err.Error(), 600)}
		log.Printf("The response is: %v", string(toJSON(verr)))
		w.WriteHeader(http.StatusBadGateway)
		w.Write(toJSON(verr))

		return
	}
	pin, err := reversePIN(fields.Pin, fields.Pan)
	if err != nil {
		log.Printf("Error in PIN reverse: %v", err)

		verr := ebs_fields.ErrorDetails{Message: "EBS Error", Code: 600, Details: generateError(fields, "Failed", err.Error(), 600)}
		log.Printf("The response is: %v", string(toJSON(verr)))
		w.WriteHeader(http.StatusBadGateway)
		w.Write(toJSON(verr))

		return
	}
	// CVV gonnaa be pin[1:]
	stripe := Stripe{PAN: fields.Pan, Amount: int(fields.TranAmount), CVV: pin[:3], ExpDate: fields.Expdate}
	payment, err := json.Marshal(&stripe)

	log.Printf("Request to Stripe: %v", string(payment))
	if err != nil {
		log.Printf("Error in PIN marshalling request: %v", err)

		verr := ebs_fields.ErrorDetails{Message: "EBS Error", Code: 600, Details: generateError(fields, "Failed", err.Error(), 600)}
		log.Printf("The response is: %v", string(toJSON(verr)))
		w.WriteHeader(http.StatusBadGateway)
		w.Write(toJSON(verr))

		return
	}

	res, err := http.Post("https://pay.int.merchant.enayatech.com/charge/", "application/json", bytes.NewBuffer(payment))
	if err != nil {
		log.Printf("Error in request to stripe: %v", err)

		verr := ebs_fields.ErrorDetails{Message: "EBS Error", Code: 600, Details: generateError(fields, "Failed", err.Error(), 600)}
		log.Printf("The response is: %v", string(toJSON(verr)))
		w.WriteHeader(http.StatusBadGateway)
		w.Write(toJSON(verr))

		return
	}

	// Enaya server could be down.
	if res.StatusCode == 500 {
		verr := ebs_fields.ErrorDetails{Message: "EBS Error", Code: 600, Details: generateError(fields, "Failed", "Visa Gateway is down", 600)}
		log.Printf("The response is: %v", string(toJSON(verr)))
		w.WriteHeader(http.StatusGatewayTimeout)
		w.Write(toJSON(verr))
		return
	}

	resData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error in marshalling stripe response: %v - Response: %v", err, string(resData))

		verr := ebs_fields.ErrorDetails{Message: "EBS Error", Code: 600, Details: generateError(fields, "Failed", err.Error(), 600)}
		log.Printf("The response is: %v", string(toJSON(verr)))
		w.WriteHeader(http.StatusBadGateway)
		w.Write(toJSON(verr))

		return
	}

	log.Printf("The status code is: %v", res.StatusCode)
	defer res.Body.Close()
	var response map[string]string
	// This should only run *if* we are testing against 400

	if res.StatusCode == http.StatusBadRequest {
		if err := json.Unmarshal(resData, &response); err != nil {
			log.Printf("Error in marshalling stripe response: %v - Response: %v", err, string(resData))

			verr := ebs_fields.ErrorDetails{Message: "EBS Error", Code: 600, Details: generateError(fields, "Failed", err.Error(), 600)}
			log.Printf("The response is: %v", string(toJSON(verr)))
			w.WriteHeader(http.StatusBadGateway)
			w.Write(toJSON(verr))

			return
		}
		if v, ok := response["messege"]; ok {
			log.Printf("the response is: %v", string(resData))
			verr := ebs_fields.ErrorDetails{Message: "EBS Error", Code: 600, Details: generateError(fields, "Failed", parseStripe(v), 600)}
			log.Printf("The response is: %v", string(toJSON(verr)))
			w.WriteHeader(http.StatusBadGateway)
			w.Write(toJSON(verr))
			return
		}

	}

	var successRes EnayaResponse
	if err := json.Unmarshal(resData, &successRes); err != nil {
		log.Printf("Error in parsing new enaya: %v", err)
		verr := ebs_fields.ErrorDetails{Message: "EBS Error", Code: 600, Details: generateError(fields, "Failed", err.Error(), 600)}
		log.Printf("The response is: %v", string(toJSON(verr)))
		w.WriteHeader(http.StatusBadGateway)
		w.Write(toJSON(verr))
		return
	}
	log.Printf("The successfull transaction is: %v", successRes)

	successfull := map[string]ebs_fields.GenericEBSResponseFields{
		"ebs_response": generateError(fields, "Successful", successRes.PaymentInfo.Status, 0),
	}
	log.Printf("The response is: %v", string(toJSON(successfull)))
	w.WriteHeader(http.StatusOK)
	w.Write(toJSON(successfull))

}

func toJSON(d interface{}) []byte {
	res, _ := json.Marshal(&d)
	return res

}

//parseStripe parses error response from Stripe
func parseStripe(res string) string {
	idx := strings.Index(res, ": ")
	if idx == -1 {
		return res
	}
	return res[idx+2:]
}

type Stripe struct {
	PAN     string `json:"card_number"`
	Amount  int    `json:"amount_in_sdg"`
	CVV     string `json:"card_cvv"`
	ExpDate string `json:"expiration_date"`
}

func reversePIN(pinblock, pan string) (string, error) {
	data := map[string]interface{}{
		"pan": pan, "tmk": TMK, "twk": TWK, "pinblock": pinblock,
	}
	req, err := json.Marshal(&data)
	if err != nil {
		log.Printf("Error in unmarshaling request: Error: %v", err)
		return "", err
	}
	res, err := http.Post("http://localhost:8008/reverse", "application/json", bytes.NewBuffer(req))
	if err != nil {
		log.Printf("Error in unmarshaling request: Error: %v", err)

		return "", err
	}

	resData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error in reading python's response: %v", err)
		return "", err
	}
	defer res.Body.Close()

	var response map[string]string
	if err := json.Unmarshal(resData, &response); err != nil {
		log.Printf("Error in reading python's response: %v", err)
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		log.Print("Failed request")
		return "", errors.New(response["message"])
	}

	return response["pin"], nil

}

type customResponse struct {
	PIN string `json:"pin"`
}

type EnayaResponse struct {
	CardNumber     string      `json:"card_number"`
	ExpirationDate string      `json:"expiration_date"`
	Amount         float32     `json:"amount_in_sdg"`
	AmountUSD      float32     `json:"amount_USD"`
	Country        string      `json:"country"`
	Currency       string      `json:"currency"`
	PaymentInfo    PaymentInfo `json:"paymentinfo"`
}

/*
	{"card_number":"4032160009749603",
	"expiration_date":"2406",
	"amount_in_sdg":250.0,
	"card_cvv":12,
	"amount_USD":12.5,
	"country":
	"United States of America","currency":"USD",
	"paymentinfo":{"id":"ch_1HqciJI3cm72eLmjOEU6YzY2",
	"captured":true,
	"created":true,
	"currency":"usd",
	"customer":null,
	"description":"International chagre to  4032160009749603 by amount  12.5",
	"paid":true,
	"payment_method":"card_1HqciJI3cm72eLmjNwrbmbZn",
	"refunded":false,
	"status":"succeeded"}}
*/

type PaymentInfo struct {
	ID            string  `json:"id"`
	Captured      bool    `json:"captured"`
	Created       bool    `json:"created"`
	Currency      string  `json:"currency"`
	Customer      *string `json:"customer"`
	Description   string  `json:"description"`
	Paid          bool    `json:"paid"`
	PaymentMethod string  `json:"payment_method"`
	Refunded      bool    `json:"refunded"`
	Status        string  `json:"status"`
}
