# SPR APNS Proxy

Proxy notifications to Apple Push Notification Service

# Description

Apple push notifications requires sending alerts using a token connected to a appleid.
SPR try not to have any centralized parts we use this proxy to forward notifications to Apple, and from there its pushed to your ios device.
If you want to you can either opt out of this in the app or setup your own notification key & talk to Apple directly.

This project is open source & alerts are encrypted from spr to the app & can only be viewed by you.
(See category: "SECRET" in this code and the ios app)

## Expected JSON
The proxy server is expecting the same json data from clients as apns
No special headers is required, use PUT or POST.

### Plaintext Notifications
```bash
curl -i 0:8000/device/1234 -X PUT \
        --data '{"aps": {"category": "PLAIN", "alert": {"title": "sample title", "body": "babody"}}}'
```

### Encrypted Notifications, in background mode
```bash
curl -i 0:8000/device/1234 -X PUT \
        --data '{"aps": {"category": "SECRET"}, "ENCRYPTED_DATA": "b64Datahere"}'
```

## Run

First setup keys:
```bash
export PRIVKEY=$(cat AuthkeyFromApple.p8 | base64)
export AUTH_KEY_ID="ABCDEFGHIJ" # should be AuthKey_NAMEHERE.p8
export TEAM_ID="1234567890" # unique teamid on apple
export TOPIC="org.supernetworks.app" # appid
```

start http proxy server:
```bash
./main
# start http server on 127.0.0.1:8000
```

test sending a notification using apns (need env keys)
```bash
./main -d 1234 -t Title -m MessageBody
```

test sending a notification using proxy on localhost:
```bash
./main -url http://0:8000 -d 1234 -t testing1234 -m lala1234
```

on localhost with data:
```bash
./main -url http://0:8000 -d 1234 --data eyJ0aXRsZSI6ICJ0ZXN0aW5nMTIzNCIsICJib2R5IjogImxhbGExMjM0In0K
```

will need to setup a notification handler to decrypt the data in the ios app.
see more in apple docs: [Modifying content in newly delivered notifications](https://developer.apple.com/documentation/usernotifications/modifying_content_in_newly_delivered_notifications/)

# Read more

Read more here on JWT tokens and sending requests with apple:
- [Sending push notifications using command-line tools](https://developer.apple.com/documentation/usernotifications/sending-push-notifications-using-command-line-tools#Send-a-Push-Notification-Using-a-Token)
- [Revoke, edit, and download keys](https://developer.apple.com/help/account/manage-keys/revoke-edit-and-download-keys)
