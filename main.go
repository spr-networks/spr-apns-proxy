package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"

	"github.com/gorilla/mux"
)

var (
	InfoLogger    *log.Logger
	WarningLogger *log.Logger
	ErrorLogger   *log.Logger
)

var PrivateKey = os.Getenv("PRIVKEY")
var AuthKeyId = os.Getenv("AUTH_KEY_ID")
var TeamId = os.Getenv("TEAM_ID")
var APNSTopic = os.Getenv("TOPIC")

//NOTE we use the same json struct format as apple here

type APNS struct {
	Aps           APNSAps `json:"aps"`
	EncryptedData string  `json:"ENCRYPTED_DATA,omitempty"`
}

type APNSAps struct {
	Category         string `json:"category,omitempty"`
	ContentAvailable int    `json:"content-available,omitempty"`
	MutableContent   int    `json:"mutable-content,omitempty"`
	// NOTE use pointer here to omit alert struct if empty
	Alert *APNSAlert `json:"alert,omitempty"`
}

type APNSAlert struct {
	Title string `json:"title,omitempty"`
	//SubTitle string `json:"subtitle"`
	Body string `json:"body,omitempty"`
}

func sendProxyNotification(host string, id string, a APNS) error {
	data, _ := json.Marshal(a)

	url := fmt.Sprintf("%s/device/%s", host, id)

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("superd request failed", err)
		return err
	}

	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("not 200 return")
	}

	return nil
}

func sendNotification(id string, alert APNSAlert) error {
	payload := APNS{Aps: APNSAps{
		ContentAvailable: 1,
		MutableContent:   1,
		Category:         "PLAIN",
		Alert:            &APNSAlert{Title: alert.Title, Body: alert.Body},
	}}

	postData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return sendNotificationData(id, postData, "alert")
}

func sendNotificationEncrypted(id string, data string) error {
	payload := APNS{
		Aps: APNSAps{
			ContentAvailable: 1,
			MutableContent:   1,
			Category:         "SECRET",
			Alert: &APNSAlert{
				Title: "Secret Message!",
				Body:  "(Encrypted)",
			},
		},
		EncryptedData: data,
	}

	postData, err := json.Marshal(payload)

	if err != nil {
		return err
	}

	return sendNotificationData(id, postData, "background")
}

func sendNotificationData(id string, data []byte, pushType string) error {
	InfoLogger.Println("data=", string(data))
	url := fmt.Sprintf("https://api.sandbox.push.apple.com/3/device/%s", id)

	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	token, err := buildToken()
	if err != nil {
		return err
	}
	//InfoLogger.Println("token=", token)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("apns-topic", APNSTopic)
	// NOTE set to background to avoid user getting two alerts
	req.Header.Add("apns-push-type", pushType) // alert or background
	auth := fmt.Sprintf("Bearer %s", token)
	req.Header.Add("Authorization", auth)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		ErrorLogger.Println("got non-200 status from apple: ", resp.StatusCode, "body=", string(body))
		//{ "reason": "TooManyProviderTokenUpdates" }
		return fmt.Errorf("apple returned %d", resp.StatusCode)
	}

	//InfoLogger.Println("<<", string(body))

	return nil
}

func notification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, fmt.Errorf("invalid method").Error(), 400)
		return
	}

	id, ok := mux.Vars(r)["id"]
	if !ok {
		http.Error(w, fmt.Errorf("missing id").Error(), 400)
		return
	}

	validId := regexp.MustCompile(`^[0-9a-fA-F]{64}$`).MatchString
	if !validId(id) {
		http.Error(w, fmt.Errorf("invalid id").Error(), 400)
		return
	}

	InfoLogger.Println("device token=", id)
	data := APNS{}

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	//TODO forward more struct items
	// currently allow setting: .alert{title,body} and .ENCRYPTED_DATA as strings

	//forward encrypted notification
	if len(data.EncryptedData) > 0 {
		err = sendNotificationEncrypted(id, data.EncryptedData)
	} else if data.Aps.Alert != nil {
		err = sendNotification(id, *data.Aps.Alert)
	} else {
		http.Error(w, "invalid json", 400)
		return
	}

	if err != nil {
		ErrorLogger.Println("failed to send notification:", err)
		http.Error(w, "failed to send notification", 400)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "done"}`)
}

func buildToken() (string, error) {
	pemBytes, _ := base64.StdEncoding.DecodeString(PrivateKey)

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return "", fmt.Errorf("failed to decode pem, need to be in base64 format")
	}

	parsed, _ := x509.ParsePKCS8PrivateKey(block.Bytes)
	privateKey := parsed.(*ecdsa.PrivateKey)

	jwt_issue_time := time.Now().Unix()
	jwt_header := fmt.Sprintf(`{ "alg": "ES256", "kid": "%s" }`, AuthKeyId)
	jwt_claims := fmt.Sprintf(`{ "iss": "%s", "iat": %d }`, TeamId, jwt_issue_time)

	jwt_header_claims := fmt.Sprintf("%s.%s",
		base64.StdEncoding.EncodeToString([]byte(jwt_header)),
		base64.StdEncoding.EncodeToString([]byte(jwt_claims)))

	hash := sha256.Sum256([]byte(jwt_header_claims))
	sig, err := ecdsa.SignASN1(rand.Reader, privateKey, hash[:])
	if err != nil {
		return "", err
	}

	valid := ecdsa.VerifyASN1(&privateKey.PublicKey, hash[:], sig)
	if !valid {
		return "", fmt.Errorf("invalid signature, should not happen")
	}

	auth_token := fmt.Sprintf("%s.%s.%s",
		base64.StdEncoding.EncodeToString([]byte(jwt_header)),
		base64.StdEncoding.EncodeToString([]byte(jwt_claims)),
		base64.StdEncoding.EncodeToString(sig))
	auth_token = strings.Replace(auth_token, "+", "-", -1)
	auth_token = strings.Replace(auth_token, "/", "_", -1)
	auth_token = strings.Replace(auth_token, "=", "", -1)

	return auth_token, nil
}

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		InfoLogger.Printf("%s %s\n", r.Method, r.URL)

		handler.ServeHTTP(w, r)
	})
}

func initServer(ip string, port int) {
	InfoLogger.Printf("starting http server on %s:%d...\n", ip, port)

	router := mux.NewRouter()
	router.HandleFunc("/device/{id}", notification).Methods("PUT", "POST")
	http.ListenAndServe(fmt.Sprintf("%s:%d", ip, port), logRequest(router))
}

func main() {
	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningLogger = log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
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
		var apns APNS

		if len(data) > 0 {
			apns = APNS{
				EncryptedData: data,
			}
		} else {
			apns = APNS{
				Aps: APNSAps{
					Category: "PLAIN", //not used
					Alert: &APNSAlert{
						Title: title,
						Body:  message,
					},
				},
			}
		}

		sendProxyNotification(proxyUrl, deviceToken, apns)
	} else if deviceToken != "" && message != "" && title != "" {
		sendNotification(deviceToken, APNSAlert{Title: title, Body: message})
	} else {
		initServer(ip, port)
	}
}
