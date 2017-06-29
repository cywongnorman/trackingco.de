package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	routing "github.com/qiangxue/fasthttp-routing"
)

func BitcoinPayDone(c *routing.Context) error {
	reason := string(c.QueryArgs().Peek("bitcoinpay-status"))

	if reason == "cancel" {
		c.Redirect("/account#log=You've cancelled! We hope you'll try again later.", http.StatusFound)
	} else if reason == "false" {
		c.Redirect("/account#log=Time is over! Do not send the payment now. Create a new payment on your account dashboard.", http.StatusFound)
	} else {
		c.Redirect("/account#log=Thank you for your payment.<br>Your account will be funded as soon as we receive the notification that your transaction was successful. If something that does not happen, contact us at fiatjaf@m.trackingco.de.", http.StatusFound)
	}

	return nil
}

func BitcoinPayIPN(c *routing.Context) error {
	incoming := struct {
		PaymentId string `json:"payment_id"`
		Ref       string `json:"reference"`
		Status    string `json:"status"`
	}{}
	err := json.Unmarshal(c.PostBody(), &incoming)
	if err != nil {
		return HTTPError{http.StatusBadRequest, "unexpected"}
	}

	res := struct {
		Data struct {
			PaymentId string `json:"payment_id"`
			Ref       string `json:"reference"`
			Status    string `json:"status"`
		} `json:"data"`
	}{}
	n, err := b.Get(BITCOINPAY+"/transaction-history/"+incoming.PaymentId, nil, &res, nil)
	if err != nil || n.Status() > 299 {
		if err == nil {
			err = fmt.Errorf("Bitcoinpay returned %d for '%s': %s",
				n.Status(), n.Url, n.RawText())
		}
		log.Print("couldn't get bitcoinpay transaction detail: " + err.Error() + ", id=" + incoming.PaymentId + ", ref=" + incoming.Ref + ", status=" + incoming.Status)
		return HTTPError{http.StatusExpectationFailed, "Couldn't get transaction detail."}
	}

	if res.Data.Ref != incoming.Ref {
		return HTTPError{http.StatusBadRequest, "wrong parameters sent."}
	}

	log.Print("bitcoinpay notification: ", incoming)

	if res.Data.Status != "confirmed" {
		// valid notification, but payment not confirmed yet
		fmt.Fprintf(c, "ok")
		return nil
	}

	parts := strings.Split(res.Data.Ref, " ") // '<email> <value>'
	email := parts[0]
	value, err := strconv.Atoi(parts[1])
	if err != nil {
		log.Print("bitcoinpay IPN invalid ref: ", res.Data.Ref)
		return HTTPError{http.StatusBadRequest, "wrong value sent."}
	}

	// add to the user's balances
	err = pg.Exec(` INSERT INTO balances (user_email, delta) VALUES (?, ?)`, email, value)
	if err != nil {
		log.Print("failed to update our database after bitcoinpay IPN. ", res.Data.Ref)
		return HTTPError{500, "failed to update our database with the payment"}
	}

	log.Print("registered payment on account ", email, ": ", value)
	fmt.Fprintf(c, "ok")
	return nil
}
