package threads

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type WebhookVerifier struct {
	appSecret   string
	verifyToken string
}

func NewWebhookVerifier(appSecret, verifyToken string) *WebhookVerifier {
	return &WebhookVerifier{
		appSecret:   appSecret,
		verifyToken: verifyToken,
	}
}

func (v *WebhookVerifier) VerifyChallenge(mode, token, challenge string) (string, bool) {
	if mode == "subscribe" && token == v.verifyToken {
		return challenge, true
	}
	return "", false
}

func (v *WebhookVerifier) VerifySignature(payload []byte, signature string) bool {
	if len(signature) < 7 || signature[:7] != "sha256=" {
		return false
	}

	expectedMAC := v.computeHMAC(payload)
	actualMAC := signature[7:]

	return hmac.Equal([]byte(expectedMAC), []byte(actualMAC))
}

func (v *WebhookVerifier) computeHMAC(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(v.appSecret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func ParseWebhookPayload(data []byte) (*WebhookPayload, error) {
	var payload WebhookPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func ExtractMentions(payload *WebhookPayload) []MentionValue {
	var mentions []MentionValue

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field == "mentions" {
				if valueData, err := json.Marshal(change.Value); err == nil {
					var mention MentionValue
					if err := json.Unmarshal(valueData, &mention); err == nil {
						mentions = append(mentions, mention)
					}
				}
			}
		}
	}

	return mentions
}
