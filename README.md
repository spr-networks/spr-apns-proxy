# SPR APNS Proxy

Proxy notifications to Apple Push Notification Service

# Description

Apple push notifications require sending alerts using a token connected to an appleid.
SPR tries not to have any centralized parts.

We use this proxy to forward notifications to Apple, and from there its pushed to your ios device.

If you want to you can either opt out of this in the app or set up your own notification key & talk to Apple directly.

Alerts are encrypted from spr to the app & can only be viewed by you.
(See category: "SECRET" in this code and the ios app)

## Expected JSON
The proxy server is expecting the same json data from clients as apns
No special headers are required, use PUT or POST.

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
#export DEV_API=1 # use api.sandbox.push.apple.com, default is api.push.apple.com
```

start http proxy server:
```bash
cd cmd && go build
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

### Testing

test sending notifications to your device when proxy is running::

```bash
export DEVICE_TOKEN="yourdevicetokenhere"
export PROXY_URL="http://localhost:8000" # default
go test -v
```

# Read more

Read more here on JWT tokens and sending requests with apple:
- [Sending push notifications using command-line tools](https://developer.apple.com/documentation/usernotifications/sending-push-notifications-using-command-line-tools#Send-a-Push-Notification-Using-a-Token)
- [Revoke, edit, and download keys](https://developer.apple.com/help/account/manage-keys/revoke-edit-and-download-keys)

## Notes on device tokens

If you install your app on a device, then get a device token by calling registerForRemoteNotifications method, the device token will not expire until the app was deleted on the device. It also doesn’t expire when your app updates to a new version or the iOS system reboots.

If you app has been deleted on a device, and then a notification is sent to the app once, then the previous device token will be marked as “not in use”.

APNs may start returning a "410 Unregistered" status for the token which has been determined to be no longer in use.
