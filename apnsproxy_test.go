package apnsproxy

//NOTE test requires proxy to run
//TODO also run with .NewServer

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
)

var proxyUrl = "http://localhost:8000"
var deviceToken = "1111111111111111111111111111111111111111111111111111111111111111"

func TestMain(t *testing.T) {
	_proxyUrl := os.Getenv("PROXY_URL")
	_deviceToken := os.Getenv("DEVICE_TOKEN")
	if _proxyUrl != "" {
		proxyUrl = _proxyUrl
	} else {
		t.Log("missing proxy url, using default")
	}

	if _deviceToken != "" {
		deviceToken = _deviceToken
	} else {
		t.Log("missing device token, using default")
	}
}

func TestDeviceAlert(t *testing.T) {
	t.Log("send plain notification: " + deviceToken)

	apns := APNS{
		Aps: APNSAps{
			Category: "PLAIN", //used for testing
			Alert: &APNSAlert{
				Title: "sprAlert",
				Body:  "AlertBody",
			},
		},
	}

	err := SendProxyNotification(proxyUrl, deviceToken, apns)

	if err != nil {
		t.Fatalf("Error: %v", err)
	}
}

//NOTE only b64 for now
func TestDeviceAlertEncrypted(t *testing.T) {
	t.Log("send encrypted notification")

	alert := APNSAlert{
		Title: "sprAlert",
		Body:  "Encrypted AlertBody",
	}

	jsonValue, _ := json.Marshal(alert)
	data := base64.StdEncoding.EncodeToString([]byte(jsonValue))

	apns := APNS{
		EncryptedData: data,
	}

	err := SendProxyNotification(proxyUrl, deviceToken, apns)

	if err != nil {
		t.Fatalf("Error: %v", err)
	}
}
