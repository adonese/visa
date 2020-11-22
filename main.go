package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"

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

func generateError(f ebs_fields.PurchaseFields, message string, code int) ebs_fields.GenericEBSResponseFields {
	return ebs_fields.GenericEBSResponseFields{
		ResponseMessage:        message,
		ResponseCode:           600,
		TerminalID:             f.TerminalID,
		ClientID:               f.ClientID,
		SystemTraceAuditNumber: 10,
		TranAmount:             f.TranAmount,
		TranDateTime:           f.TranDateTime,
		PAN:                    f.Pan,
	}
}

// WorkingKey static working key for visa purposes.
func WorkingKey(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	successfull := map[string]ebs_fields.GenericEBSResponseFields{
		"ebs_response": {
			ResponseMessage: "Approval",
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

		verr := ebs_fields.ErrorDetails{Message: "Error", Details: generateError(fields, err.Error(), 600)}
		log.Printf("The response is: %v", string(toJSON(verr)))
		w.WriteHeader(http.StatusBadGateway)
		w.Write(toJSON(verr))

		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(req, &fields); err != nil {
		log.Printf("Error in unmarshaling request: Error: %v", err)

		verr := ebs_fields.ErrorDetails{Message: "Error", Details: generateError(fields, err.Error(), 600)}
		log.Printf("The response is: %v", string(toJSON(verr)))
		w.WriteHeader(http.StatusBadGateway)
		w.Write(toJSON(verr))

		return
	}
	pin, err := reversePIN(fields.Pin, fields.Pan)
	if err != nil {
		log.Printf("Error in PIN reverse: %v", err)

		verr := ebs_fields.ErrorDetails{Message: "Error", Details: generateError(fields, err.Error(), 600)}
		log.Printf("The response is: %v", string(toJSON(verr)))
		w.WriteHeader(http.StatusBadGateway)
		w.Write(toJSON(verr))

		return
	}
	// CVV gonnaa be pin[1:]
	stripe := Stripe{PAN: fields.Pan, Amount: int(fields.TranAmount), CVV: pin[1:], ExpDate: fields.Expdate}
	payment, err := json.Marshal(&stripe)

	log.Printf("Request to Stripe: %v", string(payment))
	if err != nil {
		log.Printf("Error in PIN marshalling request: %v", err)

		verr := ebs_fields.ErrorDetails{Message: "Error", Details: generateError(fields, err.Error(), 600)}
		log.Printf("The response is: %v", string(toJSON(verr)))
		w.WriteHeader(http.StatusBadGateway)
		w.Write(toJSON(verr))

		return
	}

	res, err := http.Post("https://pay.int.merchant.enayatech.com/charge/", "application/json", bytes.NewBuffer(payment))
	if err != nil {
		log.Printf("Error in request to stripe: %v", err)

		verr := ebs_fields.ErrorDetails{Message: "Error", Details: generateError(fields, err.Error(), 600)}
		log.Printf("The response is: %v", string(toJSON(verr)))
		w.WriteHeader(http.StatusBadGateway)
		w.Write(toJSON(verr))

		return
	}

	// Enaya server could be down.
	if res.StatusCode == 500 {
		verr := ebs_fields.ErrorDetails{Message: "Error", Details: generateError(fields, "Enaya down", 600)}
		log.Printf("The response is: %v", string(toJSON(verr)))
		w.WriteHeader(http.StatusGatewayTimeout)
		w.Write(toJSON(verr))
	}

	resData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error in marshalling stripe response: %v - Response: %v", err, string(resData))

		verr := ebs_fields.ErrorDetails{Message: "Error", Details: generateError(fields, err.Error(), 600)}
		log.Printf("The response is: %v", string(toJSON(verr)))
		w.WriteHeader(http.StatusBadGateway)
		w.Write(toJSON(verr))

		return
	}

	log.Printf("The status code is: %v", res.StatusCode)
	defer res.Body.Close()
	var response map[string]string
	if err := json.Unmarshal(resData, &response); err != nil {
		log.Printf("Error in marshalling stripe response: %v - Response: %v", err, string(resData))

		verr := ebs_fields.ErrorDetails{Message: "Error", Details: generateError(fields, err.Error(), 600)}
		log.Printf("The response is: %v", string(toJSON(verr)))
		w.WriteHeader(http.StatusBadGateway)
		w.Write(toJSON(verr))

		return
	}

	if res.StatusCode != http.StatusOK {
		log.Printf("the response is: %v", string(resData))
		verr := ebs_fields.ErrorDetails{Message: "Error", Details: generateError(fields, "Failed Transaction", 600)}
		log.Printf("The response is: %v", string(toJSON(verr)))
		w.WriteHeader(http.StatusBadGateway)
		w.Write(toJSON(verr))

		return
	}
	// Successfull response here

	successfull := map[string]ebs_fields.GenericEBSResponseFields{
		"ebs_response": {
			ResponseMessage: "Approval",
			ResponseCode:    0,
		},
	}
	log.Printf("The response is: %v", string(toJSON(successfull)))
	w.WriteHeader(http.StatusOK)
	w.Write(toJSON(successfull))

}

func toJSON(d interface{}) []byte {
	res, _ := json.Marshal(&d)
	return res

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
