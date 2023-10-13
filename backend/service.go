package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/nats-io/stan.go"
	"github.com/patrickmn/go-cache"
)

const (
	pg_url string = "postgres://user:123456@localhost:5432/L0"
)

var (
	conn *pgx.Conn
	c    *cache.Cache
	err  error
	sc   stan.Conn
)

type Delivery struct {
	Name    string `json:"name" validate:"required"`
	Phone   string `json:"phone" validate:"required"`
	Zip     string `json:"zip" validate:"required"`
	City    string `json:"city" validate:"required"`
	Address string `json:"address" validate:"required"`
	Region  string `json:"region" validate:"required"`
	Email   string `json:"email" validate:"required"`
}

type Payment struct {
	Transaction   string `json:"transaction" validate:"required"`
	Request_id    string `json:"request_id"`
	Currency      string `json:"currency" validate:"required"`
	Provider      string `json:"provider" validate:"required"`
	Amount        int    `json:"amount" validate:"required"`
	Payment_dt    int    `json:"payment_dt" validate:"required"`
	Bank          string `json:"bank" validate:"required"`
	Delivery_cost int    `json:"delivery_cost" validate:"required"`
	Goods_total   int    `json:"goods_total" validate:"required"`
	Custom_fee    int    `json:"custom_fee"`
}

type Item struct {
	Chrt_id      int    `json:"chrt_id" validate:"required"`
	Track_number string `json:"track_number" validate:"required"`
	Price        int    `json:"price" validate:"required"`
	Rid          string `json:"rid" validate:"required"`
	Name         string `json:"name" validate:"required"`
	Sale         int    `json:"sale"`
	Size         string `json:"size" validate:"required"`
	Total_price  int    `json:"total_price" validate:"required"`
	Nm_id        int    `json:"nm_id" validate:"required"`
	Brand        string `json:"brand" validate:"required"`
	Status       int    `json:"status" validate:"required"`
}

type Order struct {
	Order_uid          string   `json:"order_uid" validate:"required"`
	Track_number       string   `json:"track_number" validate:"required"`
	Entry              string   `json:"entry" validate:"required"`
	Delivery           Delivery `json:"delivery" validate:"required"`
	Payment            Payment  `json:"payment" validate:"required"`
	Items              []*Item  `json:"items" validate:"required,dive,required"`
	Locale             string   `json:"locale" validate:"required"`
	Internal_signature string   `json:"internal_signature"`
	Customer_id        string   `json:"customer_id" validate:"required"`
	Delivery_service   string   `json:"delivery_service" validate:"required"`
	Shardkey           string   `json:"shardkey" validate:"required"`
	Sm_id              int      `json:"sm_id" validate:"required"`
	Date_created       string   `json:"date_created" validate:"required"`
	Oof_shard          string   `json:"oof_shard" validate:"required"`
}

func main() {
	ConnectToPG(pg_url)
	RestoreCache()
	ConnectToNatsStreaming()
	StartHTTP()

	defer conn.Close(context.Background())
	defer sc.Close()

	select {}
}

func ConnectToNatsStreaming() {
	sc, err = stan.Connect("test-cluster", "client-test1", stan.NatsURL("nats://localhost:4222"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to nats-streaming-server: %v\n", err)
		return
	}
	sc.Subscribe("foo", func(m *stan.Msg) {
		HandleMessage(m.Data)
	}, stan.DeliverAllAvailable())
}

func StartHTTP() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		order, found := FindOrder(r.URL.Query().Get("order_uid"))
		if found {
			byte, _ := json.Marshal(order)
			fmt.Fprintf(w, string(byte))
		} else {
			fmt.Fprintf(w, "Order has not be found!")
		}
	})
	http.ListenAndServe(":3001", nil)
}

func FindOrder(value string) (order *Order, found bool) {
	x, found := c.Get(value)
	if found {
		order := x.(*Order)
		found = true
		return order, found
	}
	return
}

func ConnectToPG(url string) {
	conn, err = pgx.Connect(context.Background(), url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
}

func WriteData(data []byte) {
	query := fmt.Sprintf("SELECT my_scheme.add_data('%v'::jsonb)", string(data))

	_, err = conn.Query(context.Background(), query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Data writing failed: %v\n", err)
		os.Exit(1)
	}
}

func RestoreCache() {
	c = cache.New(cache.NoExpiration, cache.NoExpiration)

	rows, err := conn.Query(context.Background(), "SELECT * FROM my_scheme.order")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		os.Exit(1)
	}

	for rows.Next() {
		var json_data []byte
		err := rows.Scan(&json_data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Data was not read %v\n", err)
			return
		}

		var order Order
		if err := json.Unmarshal(json_data, &order); err != nil {
			log.Printf("Unmarshal isn't possible: %v", err)
			os.Exit(1)
		}

		c.Set(order.Order_uid, &order, cache.DefaultExpiration)
	}

	fmt.Println("Сache has been restored!")
}

func HandleMessage(message []byte) {

	var order Order
	if err := json.Unmarshal(message, &order); err != nil {
		log.Printf("Unmarshal isn't possible: %v", err)
		return
	}

	validate := validator.New()
	if err := validate.Struct(&order); err != nil {
		errs := err.(validator.ValidationErrors)
		for _, fieldErr := range errs {
			fmt.Printf("field %s: %s\n", fieldErr.Field(), fieldErr.Tag())
		}
		return
	}
	fmt.Println("Validation succeeded!")

	_, found := c.Get(order.Order_uid)
	if found {
		fmt.Println("Order with this order_uid already exists in the database!")
	} else {
		c.Set(order.Order_uid, &order, cache.DefaultExpiration)
		WriteData(message)
		fmt.Println("Order was successfully recorded!")
	}
}
