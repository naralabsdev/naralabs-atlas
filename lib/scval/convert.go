package scval

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/stellar/go-stellar-sdk/xdr"
)

func (p *Parser) parseBase64Strict(base64XDR string) (json.RawMessage, error) {
	var val xdr.ScVal
	if err := xdr.SafeUnmarshalBase64(base64XDR, &val); err != nil {
		return nil, fmt.Errorf("unmarshal scval xdr: %w", err)
	}

	typed, err := p.valToTagged(val)
	if err != nil {
		return nil, err
	}

	out, err := json.Marshal(typed)
	if err != nil {
		return nil, fmt.Errorf("marshal tagged scval: %w", err)
	}
	return out, nil
}

func (p *Parser) valToTagged(val xdr.ScVal) (any, error) {
	switch val.Type {
	case xdr.ScValTypeScvBool:
		return tagged("bool", *val.B), nil
	case xdr.ScValTypeScvVoid:
		return tagged("void", nil), nil
	case xdr.ScValTypeScvU32:
		return tagged("u32", uint32(*val.U32)), nil
	case xdr.ScValTypeScvI32:
		return tagged("i32", int32(*val.I32)), nil
	case xdr.ScValTypeScvU64:
		return tagged("u64", uint64(*val.U64)), nil
	case xdr.ScValTypeScvI64:
		return tagged("i64", int64(*val.I64)), nil
	case xdr.ScValTypeScvTimepoint:
		return tagged("timepoint", uint64(*val.Timepoint)), nil
	case xdr.ScValTypeScvDuration:
		return tagged("duration", uint64(*val.Duration)), nil
	case xdr.ScValTypeScvU128:
		return tagged("u128", u128Decimal(*val.U128)), nil
	case xdr.ScValTypeScvI128:
		return tagged("i128", i128Decimal(*val.I128)), nil
	case xdr.ScValTypeScvU256:
		return tagged("u256", u256Decimal(*val.U256)), nil
	case xdr.ScValTypeScvI256:
		return tagged("i256", i256Decimal(*val.I256)), nil
	case xdr.ScValTypeScvBytes:
		return tagged("bytes", hex.EncodeToString(*val.Bytes)), nil
	case xdr.ScValTypeScvString:
		return tagged("string", string(*val.Str)), nil
	case xdr.ScValTypeScvSymbol:
		return tagged("symbol", string(*val.Sym)), nil
	case xdr.ScValTypeScvAddress:
		addr, err := val.Address.String()
		if err != nil {
			return nil, fmt.Errorf("render address: %w", err)
		}
		return tagged("address", addr), nil
	case xdr.ScValTypeScvVec:
		return p.vecToTagged(val)
	case xdr.ScValTypeScvMap:
		return p.mapToTagged(val)
	case xdr.ScValTypeScvContractInstance:
		return p.contractInstanceToTagged(val)
	case xdr.ScValTypeScvLedgerKeyNonce:
		return tagged("ledger_key_nonce", map[string]any{
			"nonce": int64(val.NonceKey.Nonce),
		}), nil
	case xdr.ScValTypeScvExecutableTag:
		return tagged("executable_tag", string(*val.ExecutableTag)), nil
	case xdr.ScValTypeScvError:
		return p.errorToTagged(val), nil
	default:
		raw, err := xdr.MarshalBase64(val)
		if err != nil {
			return nil, fmt.Errorf("re-encode scval type %s: %w", val.Type, err)
		}
		return opaqueDocument("unsupported_scval_type", val.Type.String(), raw), nil
	}
}

func tagged(kind string, value any) map[string]any {
	return map[string]any{kind: value}
}

func (p *Parser) vecToTagged(val xdr.ScVal) (any, error) {
	if val.Vec == nil || *val.Vec == nil {
		return tagged("vec", []any{}), nil
	}

	vec := *val.Vec
	items := make([]any, 0, len(*vec))
	for _, item := range *vec {
		decoded, err := p.valToTagged(item)
		if err != nil {
			return nil, err
		}
		items = append(items, decoded)
	}
	return tagged("vec", items), nil
}

func (p *Parser) mapToTagged(val xdr.ScVal) (any, error) {
	if val.Map == nil || *val.Map == nil {
		return tagged("map", map[string]any{"entries": []any{}}), nil
	}

	entries, err := p.scMapEntries(**val.Map)
	if err != nil {
		return nil, err
	}
	return tagged("map", map[string]any{"entries": entries}), nil
}

func (p *Parser) scMapEntries(m xdr.ScMap) ([]any, error) {
	entries := make([]any, 0, len(m))
	for _, entry := range m {
		key, err := p.valToTagged(entry.Key)
		if err != nil {
			return nil, err
		}
		value, err := p.valToTagged(entry.Val)
		if err != nil {
			return nil, err
		}
		entries = append(entries, map[string]any{"k": key, "v": value})
	}
	return entries, nil
}

func (p *Parser) contractInstanceToTagged(val xdr.ScVal) (any, error) {
	inst := *val.Instance
	exec, err := p.executableToTagged(inst.Executable)
	if err != nil {
		return nil, err
	}

	storage := []any{}
	if inst.Storage != nil {
		storage, err = p.scMapEntries(*inst.Storage)
		if err != nil {
			return nil, err
		}
	}

	return tagged("contract_instance", map[string]any{
		"executable": exec,
		"storage":    storage,
	}), nil
}

func (p *Parser) executableToTagged(exec xdr.ContractExecutable) (any, error) {
	switch exec.Type {
	case xdr.ContractExecutableTypeContractExecutableWasm:
		return map[string]any{"wasm_hash": exec.WasmHash.HexString()}, nil
	case xdr.ContractExecutableTypeContractExecutableStellarAsset:
		return map[string]any{"stellar_asset": true}, nil
	case xdr.ContractExecutableTypeContractExecutableExternalRef:
		owner, err := exec.ExternalRef.ExecutableOwner.String()
		if err != nil {
			return nil, fmt.Errorf("render external ref owner: %w", err)
		}
		return map[string]any{
			"external_ref": map[string]any{
				"owner": owner,
				"tag":   string(exec.ExternalRef.Tag),
			},
		}, nil
	default:
		raw, err := xdr.MarshalBase64(exec)
		if err != nil {
			return nil, fmt.Errorf("re-encode executable: %w", err)
		}
		return opaqueDocument("unsupported_executable_type", exec.Type.String(), raw), nil
	}
}

func (p *Parser) errorToTagged(val xdr.ScVal) any {
	payload := map[string]any{"type": val.Error.Type.String()}
	if val.Error.ContractCode != nil {
		payload["contract_code"] = uint32(*val.Error.ContractCode)
	}
	if val.Error.Code != nil {
		payload["code"] = val.Error.Code.String()
	}
	return tagged("error", payload)
}
