package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/valyala/fasthttp"
)

func handleStrikeWebhook(c *fasthttp.RequestCtx) {
	log.Info().Msg("strike webhook")
	c.SuccessString("text/plain", "ok")

	var ee StrikeEvent
	err := json.Unmarshal(c.PostBody(), &ee)
	if err != nil {
		log.Warn().Err(err).Msg("error decoding webhook from strike")
		return
	}

	res, err := pg.Exec(
		`UPDATE payments SET paid_at = 'now'::timestamp, amount = $1 WHERE id = $2`,
		ee.Data.AmountSatoshi, ee.Data.Id)
	if err != nil {
		log.Warn().Err(err).Msg("error updating strike charge on db")
		return
	}

	nrows, _ := res.RowsAffected()
	log.Info().
		Int64("rows-affected", nrows).
		Str("id", ee.Data.Id).
		Int64("amt", ee.Data.AmountSatoshi).
		Str("desc", ee.Data.Description).
		Msg("webhook updated payment")
}

func acquireStrikeRequest() *fasthttp.Request {
	auth := base64.StdEncoding.EncodeToString([]byte(s.StrikeAPIKey + ":"))
	req := fasthttp.AcquireRequest()
	req.Header.Set("Authorization", "Basic "+auth)
	return req
}

func reuseCharge(userId string, amount int) (charge StrikeCharge, err error) {
	var p Payment
	err = pg.Get(&p, `
SELECT * FROM payments
WHERE user_id = $1 AND amount = $2 AND paid_at IS NULL
ORDER BY created_at DESC
LIMIT 1
    `)
	if err != nil {
		return
	}

	req := acquireStrikeRequest()
	req.SetRequestURI("https://api.dev.strike.acinq.co/api/v1/charges/" + p.Id)
	req.Header.SetMethod("GET")
	resp := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	err = fasthttp.Do(req, resp)
	if err == nil && resp.Header.StatusCode() >= 300 {
		err = errors.New(string(resp.Body()))
	}
	if err != nil {
		return
	}

	err = json.Unmarshal(resp.Body(), &charge)
	return
}

func createCharge(userId string, amount int) (charge StrikeCharge, err error) {
	if charge, err = reuseCharge(userId, amount); err == nil {
		return charge, err
	}

	args := fasthttp.AcquireArgs()
	args.Add("description", userId)
	args.Add("amount", strconv.Itoa(amount))
	args.Add("currency", "btc")

	req := acquireStrikeRequest()
	req.SetRequestURI("https://api.strike.acinq.co/api/v1/charges")
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/x-www-form-urlencoded")
	_, err = args.WriteTo(req.BodyWriter())
	if err != nil {
		return
	}

	resp := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseArgs(args)
	defer fasthttp.ReleaseResponse(resp)

	err = fasthttp.Do(req, resp)
	if err == nil && resp.Header.StatusCode() >= 300 {
		err = errors.New(string(resp.Body()))
	}
	if err != nil {
		return
	}

	err = json.Unmarshal(resp.Body(), &charge)
	if err != nil {
		return
	}

	_, err = pg.Exec(`
INSERT INTO payments (user_id, id, amount, created_at, paid_at)
VALUES (
  $1, $2, $3,
  to_timestamp($4),
  CASE WHEN $5 THEN 'now'::timestamp ELSE NULL END
)
    `, userId, charge.Id, charge.AmountSatoshi, charge.Created/1000, charge.Paid)
	return
}

type StrikeEvent struct {
	Object string       `json:"object"`
	Data   StrikeCharge `json:"data"`
}

type StrikeCharge struct {
	Id             string `json:"id"`
	Object         string `json:"object"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	AmountSatoshi  int64  `json:"amount_satoshi"`
	PaymentHash    string `json:"payment_hash"`
	PaymentRequest string `json:"payment_request"`
	Description    string `json:"description"`
	Paid           bool   `json:"paid"`
	Created        int64  `json:"created"`
	Updated        int64  `json:"updated"`
}
