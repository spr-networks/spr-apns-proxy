package main

import (
"flag"
"log"
"os"

apnsproxy "github.com/spr-networks/spr-apns-proxy"
)

var (
    ErrorLogger   *log.Logger
)

var PrivateKey = os.Getenv("PRIVKEY")
var AuthKeyId = os.Getenv("AUTH_KEY_ID")
var TeamId = os.Getenv("TEAM_ID")
var APNSTopic = os.Getenv("TOPIC")

func main() {
	ErrorLogger = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	var deviceToken, message, title string

	flag.StringVar(&deviceToken, "d", "", "device token")
	flag.StringVar(&message, "m", "", "alert body message")
	flag.StringVar(&title, "t", "", "alert title")
	var proxyUrl, data string
	flag.StringVar(&proxyUrl, "url", "", "test sending using proxy")
	flag.StringVar(&data, "data", "", "send base64-encoded payload as SECRET category")

	var ip string
	var port int
	flag.StringVar(&ip, "ip", "127.0.0.1", "listen on ip")
	flag.IntVar(&port, "port", 8000, "listen on port")
	flag.Parse()

	// need these if either server or test send to apns, not for proxy send
	if len(proxyUrl) == 0 {
		if PrivateKey == "" {
			ErrorLogger.Println("missing PRIVKEY")
			return
		}

		if AuthKeyId == "" {
			ErrorLogger.Println("missing AUTH_KEY_ID")
			return
		}

		if TeamId == "" {
			ErrorLogger.Println("missing TEAM_ID")
			return
		}

		if APNSTopic == "" {
			ErrorLogger.Println("missing TOPIC")
			return
		}
	}

	if len(proxyUrl) > 0 {
		var apns apnsproxy.APNS

		if len(data) > 0 {
			apns = apnsproxy.APNS{
				EncryptedData: data,
			}
		} else {
			apns = apnsproxy.APNS{
				Aps: apnsproxy.APNSAps{
					Category: "PLAIN", //not used
					Alert: &apnsproxy.APNSAlert{
						Title: title,
						Body:  message,
					},
				},
			}
		}

		apnsproxy.SendProxyNotification(proxyUrl, deviceToken, apns)
	} else if deviceToken != "" && message != "" && title != "" {
		apnsproxy.SendNotification(deviceToken, apnsproxy.APNSAlert{Title: title, Body: message})
	} else {
		apnsproxy.NewServer(ip, port)
	}
}
