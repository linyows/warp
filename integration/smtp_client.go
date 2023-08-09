package integration

import (
	"fmt"
	"log"
	"net/smtp"
)

func SendEmail() error {
	c, err := smtp.Dial(ip + ":" + port)
	if err != nil {
		log.Println("smtp dial error")
		return err
	}
	if err := c.Mail("alice@example.test"); err != nil {
		log.Println("smtp mail error")
		return err
	}
	if err := c.Rcpt("bob@example.local"); err != nil {
		log.Println("smtp rcpt error")
		return err
	}
	wc, err := c.Data()
	if err != nil {
		log.Println("smtp data error")
		return err
	}
	_, err = fmt.Fprintf(wc, "This is the email body")
	if err != nil {
		log.Println("smtp data print error")
		return err
	}
	if err = wc.Close(); err != nil {
		log.Println("smtp close print error")
		return err
	}
	if err = c.Quit(); err != nil {
		log.Println("smtp quit error")
		return err
	}
	return nil
}
