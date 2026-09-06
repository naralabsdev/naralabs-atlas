package scval

import (
	"encoding/json"
	"fmt"
)

func opaquePayload(reason, base64XDR string, err error) json.RawMessage {
	payload, marshalErr := json.Marshal(opaqueDocument(reason, "", base64XDR, err.Error()))
	if marshalErr != nil {
		return json.RawMessage(fmt.Sprintf(`{"opaque":{"reason":"marshal_failed","encoding":"base64_xdr","data":%q}}`, base64XDR))
	}
	return payload
}

func opaqueDocument(reason, scType, base64XDR string, detail ...string) map[string]any {
	doc := map[string]any{
		"opaque": map[string]any{
			"reason":   reason,
			"encoding": "base64_xdr",
			"data":     base64XDR,
		},
	}
	if scType != "" {
		doc["opaque"].(map[string]any)["sc_type"] = scType
	}
	if len(detail) > 0 && detail[0] != "" {
		doc["opaque"].(map[string]any)["detail"] = detail[0]
	}
	return doc
}

func opaqueType(typeName, base64XDR string) json.RawMessage {
	payload, err := json.Marshal(opaqueDocument("unsupported_scval_type", typeName, base64XDR))
	if err != nil {
		return json.RawMessage(`{"opaque":{"reason":"unsupported_scval_type"}}`)
	}
	return payload
}
