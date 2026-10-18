package loopia_test

import (
	"context"
	"errors"
	"fmt"
	"log"

	loopia "github.com/Sajfer/loopia-go"
)

func Example() {
	ctx := context.Background()
	client := loopia.New("user@loopiaapi", "password")

	record := loopia.Record{Type: "A", TTL: 3600, Value: "192.0.2.10"}
	if err := client.AddZoneRecord(ctx, "example.com", "www", &record); err != nil {
		log.Fatal(err)
	}
	fmt.Println("created record", record.ID)

	record.Value = "192.0.2.20"
	if err := client.UpdateZoneRecord(ctx, "example.com", "www", record); err != nil {
		log.Fatal(err)
	}
}

func ExampleClient_ForCustomer() {
	reseller := loopia.New("reseller@loopiaapi", "password")

	domains, err := reseller.ForCustomer("C12345").GetDomains(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	for _, d := range domains {
		fmt.Println(d.Name, d.ExpirationDate)
	}
}

func ExampleStatusError() {
	client := loopia.New("user@loopiaapi", "wrong password")

	err := client.AddSubdomain(context.Background(), "example.com", "www")
	switch {
	case errors.Is(err, loopia.ErrAuth):
		fmt.Println("check your LoopiaAPI credentials")
	case errors.Is(err, loopia.ErrRateLimited),
		errors.Is(err, &loopia.FaultError{Code: loopia.FaultRateLimited}):
		fmt.Println("slow down: at most 60 calls per minute")
	case err != nil:
		log.Fatal(err)
	}
}
