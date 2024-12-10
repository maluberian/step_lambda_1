package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	netMail "net/mail"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

const (
	OFFERED  string = "OFFERED"
	AGREED   string = "AGREED"
	DECLINED string = "DECLINED"
)

type BoopyOffer struct {
	Id            int32   `json:"id"`
	SenderEmail   string  `json:"sender"`
	SellerEmail   string  `json:"seller"`
	BuyerEmail    string  `json:"buyer"`
	OfferAmount   float32 `json:"offerAmount"`
	RequestAmount float32 `json:"requestAmount"`
	Status        string  `json:"status"`
}

func validateOffer(offer BoopyOffer) error {
	_, err := netMail.ParseAddress(offer.BuyerEmail)
	if err != nil {
		return errors.New("invalid buyer email address")
	}
	_, err = netMail.ParseAddress(offer.SellerEmail)
	if err != nil {
		return errors.New("invalid seller email address")
	}
	if offer.OfferAmount <= 0 {
		return errors.New("offer ammount must be greater than zero")
	}
	return nil
}

func HandleRequest(ctx context.Context, event json.RawMessage) error {
	sendgridApiKey := os.Getenv("SENDGRID_API_KEY")
	if sendgridApiKey == "" {
		return errors.New("SENDGRID_API_KEY is not set")
	}

	if event == nil {
		return errors.New("empty event")
	}

	var txn BoopyOffer
	if err := json.Unmarshal(event, &txn); err != nil {
		return err
	}

	err := validateOffer(txn)
	if err != nil {
		return err
	}

	from := mail.NewEmail("RedSpur", txn.SenderEmail)
	to := mail.NewEmail("RedSpur Seller", txn.SellerEmail)
	subject := fmt.Sprintf("New offer from %s", txn.SellerEmail)
	htmlBody := fmt.Sprintf("Transaction <strong>%d</strong>: <strong>%s</strong> offers <mark>%f</mark>!", txn.Id, txn.SellerEmail, txn.OfferAmount)
	textBody := fmt.Sprintf("Transaction %d: %s offers %f!", txn.Id, txn.SellerEmail, txn.OfferAmount)
	message := mail.NewSingleEmail(from, subject, to, textBody, htmlBody)
	client := sendgrid.NewSendClient(sendgridApiKey)
	response, err := client.Send(message)
	if err != nil {
		return err
	}
	log.Printf("Email Response: send responded with %d (%s)", response.StatusCode, response.Body)

	return nil
}

func main() {
	lambda.Start(HandleRequest)
}
