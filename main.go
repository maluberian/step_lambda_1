package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	netMail "net/mail"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

type S3Event struct {
	Version    string    `json:"version"`
	Id         string    `json:"id"`
	DetailType string    `json:"detail-type"`
	Source     string    `json:"source"`
	Account    string    `json:"account"`
	Time       time.Time `json:"time"`
	Region     string    `json:"region"`
	Resources  []string  `json:"resources"`
	Detail     struct {
		Version string `json:"version"`
		Bucket  struct {
			Name string `json:"name"`
		} `json:"bucket"`
		Object struct {
			Key       string `json:"key"`
			Sequencer string `json:"sequencer"`
		} `json:"object"`
		RequestId       string `json:"request-id"`
		Requester       string `json:"requester"`
		SourceIpAddress string `json:"source-ip-address"`
		Reason          string `json:"reason"`
		DeletionType    string `json:"deletion-type"`
	} `json:"detail"`
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
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return errors.New(fmt.Sprintf("error loading AWS config, %v", err.Error()))
	}

	sendgridApiKey := os.Getenv("SENDGRID_API_KEY")
	if sendgridApiKey == "" {
		return errors.New("SENDGRID_API_KEY is not set")
	}

	if event == nil {
		return errors.New("empty event")
	}

	j, err := json.Marshal(&event)
	if err != nil {
		return err
	}
	log.Printf("Event: %s", j)

	var s3Event S3Event
	if err := json.Unmarshal(event, &s3Event); err != nil {
		return err
	}
	log.Printf("Event: %v", s3Event)

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.Region = s3Event.Region
	})
	rsp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3Event.Detail.Bucket.Name),
		Key:    aws.String(s3Event.Detail.Object.Key),
	})
	if err != nil {
		return err
	}
	defer func() {
		err := rsp.Body.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	bdy, err := io.ReadAll(rsp.Body)
	if err != nil {
		return err
	}
	log.Printf("S3 Document: %s", string(bdy))

	var txn = BoopyOffer{}
	err = json.Unmarshal(bdy, &txn)
	if err != nil {
		return err
	}

	err = validateOffer(txn)
	if err != nil {
		return err
	}

	from := mail.NewEmail("RedSpur", txn.SenderEmail)
	to := mail.NewEmail("RedSpur Seller", txn.SellerEmail)
	subject := fmt.Sprintf("New offer from %s", txn.BuyerEmail)
	htmlBody := fmt.Sprintf("Transaction <strong>%d</strong>: <strong>%s</strong> offers <mark>%f</mark>!", txn.Id, txn.BuyerEmail, txn.OfferAmount)
	textBody := fmt.Sprintf("Transaction %d: %s offers %f!", txn.Id, txn.BuyerEmail, txn.OfferAmount)
	message := mail.NewSingleEmail(from, subject, to, textBody, htmlBody)
	sgClient := sendgrid.NewSendClient(sendgridApiKey)
	response, err := sgClient.Send(message)
	if err != nil {
		return err
	}
	log.Printf("Email Response: send responded with %d (%s)", response.StatusCode, response.Body)

	return nil
}

func main() {
	lambda.Start(HandleRequest)
}
