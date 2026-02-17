package so

import (
	"bytes"
	"debug/elf"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/perbu/vclparser/pkg/vcc"
)

const (
	jsonSymbolName      = "Vmod_Json"
	jsonStartPattern    = "[[\"$VMOD\""
	jsonLegacyStartMark = "VMOD_JSON_SPEC\x02"
	jsonLegacyEndMark   = byte(0x03)
)

// LoadModuleFromSO loads VMOD interface metadata from a Linux ELF shared object.
// It extracts embedded VMOD JSON and converts it to the internal vcc.Module model.
func LoadModuleFromSO(filename string) (*vcc.Module, error) {
	soFile, err := elf.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s as ELF shared object: %w", filename, err)
	}
	defer soFile.Close()

	moduleName := inferModuleName(filename, soFile)
	if moduleName == "" {
		return nil, fmt.Errorf("failed to infer module name from %s", filename)
	}

	blob, err := extractVMODJSON(filename, soFile)
	if err != nil {
		return nil, err
	}

	stanzas, err := decodeJSONStanzas(blob)
	if err != nil {
		return nil, fmt.Errorf("failed to decode VMOD JSON from %s: %w", filename, err)
	}

	module, err := stanzasToModule(moduleName, stanzas)
	if err != nil {
		return nil, fmt.Errorf("failed to parse VMOD JSON interface from %s: %w", filename, err)
	}

	return module, nil
}

func inferModuleName(filename string, soFile *elf.File) string {
	if name := moduleNameFromDataSymbol(soFile); name != "" {
		return strings.ToLower(name)
	}

	base := filepath.Base(filename)
	if strings.HasPrefix(base, "libvmod_") && strings.HasSuffix(base, ".so") {
		base = strings.TrimPrefix(base, "libvmod_")
		base = strings.TrimSuffix(base, ".so")
		return strings.ToLower(base)
	}

	base = strings.TrimSuffix(base, filepath.Ext(base))
	return strings.ToLower(base)
}

func moduleNameFromDataSymbol(soFile *elf.File) string {
	dynSyms, err := soFile.DynamicSymbols()
	if err == nil {
		for _, sym := range dynSyms {
			if strings.HasPrefix(sym.Name, "Vmod_") && strings.HasSuffix(sym.Name, "_Data") {
				return strings.TrimSuffix(strings.TrimPrefix(sym.Name, "Vmod_"), "_Data")
			}
		}
	}

	syms, err := soFile.Symbols()
	if err != nil {
		return ""
	}
	for _, sym := range syms {
		if strings.HasPrefix(sym.Name, "Vmod_") && strings.HasSuffix(sym.Name, "_Data") {
			return strings.TrimSuffix(strings.TrimPrefix(sym.Name, "Vmod_"), "_Data")
		}
	}
	return ""
}

func extractVMODJSON(filename string, soFile *elf.File) ([]byte, error) {
	blob, err := extractVMODJSONFromSymbol(soFile, jsonSymbolName)
	if err == nil {
		return blob, nil
	}

	// Fallback for binaries where symbol metadata is unavailable/stripped:
	// scan the raw bytes for embedded JSON content.
	raw, readErr := os.ReadFile(filename)
	if readErr != nil {
		return nil, fmt.Errorf("failed to extract VMOD JSON from %s: %v (and fallback read failed: %w)", filename, err, readErr)
	}

	blob, extractErr := extractVMODJSONFromBytes(raw)
	if extractErr != nil {
		return nil, fmt.Errorf("failed to extract VMOD JSON from %s: %v (and raw scan failed: %w)", filename, err, extractErr)
	}

	return blob, nil
}

func extractVMODJSONFromSymbol(soFile *elf.File, symbolName string) ([]byte, error) {
	symbols, err := soFile.Symbols()
	if err != nil {
		return nil, fmt.Errorf("failed to read symbol table: %w", err)
	}

	var vmodJSONSymbol *elf.Symbol
	for i := range symbols {
		if symbols[i].Name == symbolName {
			vmodJSONSymbol = &symbols[i]
			break
		}
	}

	if vmodJSONSymbol == nil {
		return nil, fmt.Errorf("symbol %s not found", symbolName)
	}

	data, err := readSymbolData(soFile, *vmodJSONSymbol)
	if err != nil {
		return nil, fmt.Errorf("failed to read symbol %s: %w", symbolName, err)
	}

	blob, err := extractVMODJSONFromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("invalid data in symbol %s: %w", symbolName, err)
	}
	return blob, nil
}

func readSymbolData(soFile *elf.File, symbol elf.Symbol) ([]byte, error) {
	if symbol.Section == elf.SHN_UNDEF {
		return nil, fmt.Errorf("symbol %s is undefined", symbol.Name)
	}
	if int(symbol.Section) >= len(soFile.Sections) {
		return nil, fmt.Errorf("invalid section index %d", symbol.Section)
	}

	section := soFile.Sections[symbol.Section]
	sectionData, err := section.Data()
	if err != nil {
		return nil, fmt.Errorf("failed to read section data: %w", err)
	}

	if symbol.Value < section.Addr {
		return nil, fmt.Errorf("symbol value %#x is before section start %#x", symbol.Value, section.Addr)
	}
	offset := symbol.Value - section.Addr
	if offset > uint64(len(sectionData)) {
		return nil, fmt.Errorf("symbol offset %d is outside section", offset)
	}

	end := uint64(len(sectionData))
	if symbol.Size > 0 && offset+symbol.Size <= end {
		end = offset + symbol.Size
	}

	value := sectionData[offset:end]
	return bytes.TrimRight(value, "\x00"), nil
}

func extractVMODJSONFromBytes(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	if blob, ok := extractLegacyJSON(data); ok {
		return blob, nil
	}

	searchStart := 0
	for {
		idx := bytes.Index(data[searchStart:], []byte(jsonStartPattern))
		if idx < 0 {
			return nil, fmt.Errorf("VMOD JSON start marker not found")
		}
		start := searchStart + idx

		decoder := json.NewDecoder(bytes.NewReader(data[start:]))
		decoder.UseNumber()
		var stanzas []any
		if err := decoder.Decode(&stanzas); err == nil {
			end := start + int(decoder.InputOffset())
			if end > start && end <= len(data) {
				return bytes.TrimRight(data[start:end], "\x00"), nil
			}
		}

		searchStart = start + 1
	}
}

func extractLegacyJSON(data []byte) ([]byte, bool) {
	start := bytes.Index(data, []byte(jsonLegacyStartMark))
	if start < 0 {
		return nil, false
	}

	start += len(jsonLegacyStartMark)
	endRel := bytes.IndexByte(data[start:], jsonLegacyEndMark)
	if endRel < 0 {
		return nil, false
	}

	blob := data[start : start+endRel]
	return bytes.TrimRight(blob, "\x00"), true
}

func decodeJSONStanzas(blob []byte) ([]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(blob))
	decoder.UseNumber()

	var stanzas []any
	if err := decoder.Decode(&stanzas); err != nil {
		return nil, err
	}

	return stanzas, nil
}

func stanzasToModule(moduleName string, stanzas []any) (*vcc.Module, error) {
	module := &vcc.Module{
		Name:      moduleName,
		Version:   3,
		ABI:       "so-json",
		Functions: []vcc.Function{},
		Objects:   []vcc.Object{},
		Events:    []vcc.Event{},
	}

	var lastFunction *vcc.Function

	for idx, rawStanza := range stanzas {
		stanza, ok := rawStanza.([]any)
		if !ok || len(stanza) == 0 {
			continue
		}

		tag, _ := asString(stanza[0])
		switch tag {
		case "$VMOD":
			if len(stanza) > 1 {
				if jsonVersion, ok := asString(stanza[1]); ok && jsonVersion != "" {
					module.ABI = "so-json:" + jsonVersion
				}
			}

		case "$FUNC":
			function, err := parseFunctionStanza(stanza)
			if err != nil {
				return nil, fmt.Errorf("stanza %d ($FUNC): %w", idx, err)
			}
			module.Functions = append(module.Functions, function)
			lastFunction = &module.Functions[len(module.Functions)-1]

		case "$OBJ":
			object, err := parseObjectStanza(stanza)
			if err != nil {
				return nil, fmt.Errorf("stanza %d ($OBJ): %w", idx, err)
			}
			module.Objects = append(module.Objects, object)
			lastFunction = nil

		case "$EVENT":
			event := parseEventStanza(stanza)
			if event.Name != "" {
				module.Events = append(module.Events, event)
			}
			lastFunction = nil

		case "$RESTRICT":
			if lastFunction != nil {
				lastFunction.Restrictions = append(lastFunction.Restrictions, parseRestrictions(stanza)...)
			}
		}
	}

	return module, nil
}

func parseFunctionStanza(stanza []any) (vcc.Function, error) {
	if len(stanza) < 3 {
		return vcc.Function{}, fmt.Errorf("invalid function stanza length: %d", len(stanza))
	}

	name, ok := asString(stanza[1])
	if !ok || name == "" {
		return vcc.Function{}, fmt.Errorf("invalid function name")
	}

	signature, ok := stanza[2].([]any)
	if !ok {
		return vcc.Function{}, fmt.Errorf("invalid function signature")
	}

	returnType, parameters, err := parseSignature(signature)
	if err != nil {
		return vcc.Function{}, err
	}

	return vcc.Function{
		Name:         name,
		ReturnType:   returnType,
		Parameters:   parameters,
		Examples:     []string{},
		Restrictions: []string{},
	}, nil
}

func parseObjectStanza(stanza []any) (vcc.Object, error) {
	if len(stanza) < 2 {
		return vcc.Object{}, fmt.Errorf("invalid object stanza length: %d", len(stanza))
	}

	name, ok := asString(stanza[1])
	if !ok || name == "" {
		return vcc.Object{}, fmt.Errorf("invalid object name")
	}

	object := vcc.Object{
		Name:        name,
		Constructor: []vcc.Parameter{},
		Methods:     []vcc.Method{},
		Examples:    []string{},
	}

	var lastMethod *vcc.Method

	for i := 2; i < len(stanza); i++ {
		child, ok := stanza[i].([]any)
		if !ok || len(child) == 0 {
			continue
		}

		tag, _ := asString(child[0])
		switch tag {
		case "$INIT":
			signature, err := getSignatureFromObjectChild(child)
			if err != nil {
				return vcc.Object{}, fmt.Errorf("invalid $INIT signature: %w", err)
			}
			_, parameters, err := parseSignature(signature)
			if err != nil {
				return vcc.Object{}, fmt.Errorf("invalid $INIT parameters: %w", err)
			}
			object.Constructor = parameters

		case "$METHOD":
			if len(child) < 3 {
				return vcc.Object{}, fmt.Errorf("invalid $METHOD stanza length: %d", len(child))
			}

			methodName, ok := asString(child[1])
			if !ok || methodName == "" {
				return vcc.Object{}, fmt.Errorf("invalid $METHOD name")
			}

			signature, ok := child[2].([]any)
			if !ok {
				return vcc.Object{}, fmt.Errorf("invalid $METHOD signature")
			}

			returnType, parameters, err := parseSignature(signature)
			if err != nil {
				return vcc.Object{}, fmt.Errorf("invalid $METHOD signature for %s: %w", methodName, err)
			}

			method := vcc.Method{
				Name:         methodName,
				ReturnType:   returnType,
				Parameters:   parameters,
				Examples:     []string{},
				Restrictions: []string{},
			}
			object.Methods = append(object.Methods, method)
			lastMethod = &object.Methods[len(object.Methods)-1]

		case "$RESTRICT":
			if lastMethod != nil {
				lastMethod.Restrictions = append(lastMethod.Restrictions, parseRestrictions(child)...)
			}
		}
	}

	return object, nil
}

func getSignatureFromObjectChild(child []any) ([]any, error) {
	if len(child) < 2 {
		return nil, fmt.Errorf("invalid object child length: %d", len(child))
	}

	signature, ok := child[len(child)-1].([]any)
	if !ok {
		return nil, fmt.Errorf("signature is not an array")
	}

	return signature, nil
}

func parseEventStanza(stanza []any) vcc.Event {
	if len(stanza) < 2 {
		return vcc.Event{}
	}

	name, ok := asString(stanza[1])
	if !ok || name == "" {
		return vcc.Event{}
	}

	if parts := strings.Split(name, "."); len(parts) > 0 {
		name = parts[len(parts)-1]
	}

	return vcc.Event{Name: name}
}

func parseRestrictions(stanza []any) []string {
	if len(stanza) < 2 {
		return nil
	}

	entries, ok := stanza[1].([]any)
	if !ok {
		return nil
	}

	restrictions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if restriction, ok := asString(entry); ok && restriction != "" {
			restrictions = append(restrictions, restriction)
		}
	}

	return restrictions
}

func parseSignature(signature []any) (vcc.VCCType, []vcc.Parameter, error) {
	if len(signature) == 0 {
		return "", nil, fmt.Errorf("empty signature")
	}

	returnType, err := parseReturnType(signature[0])
	if err != nil {
		return "", nil, err
	}

	parameters := make([]vcc.Parameter, 0)
	argStart := 3
	if len(signature) > 2 {
		if _, isArg := signature[2].([]any); isArg {
			argStart = 2
		}
	}

	for i := argStart; i < len(signature); i++ {
		argDef, ok := signature[i].([]any)
		if !ok || len(argDef) == 0 {
			continue
		}

		param, err := parseParameter(argDef)
		if err != nil {
			return "", nil, fmt.Errorf("invalid parameter %d: %w", i-argStart+1, err)
		}
		parameters = append(parameters, param)
	}

	return returnType, parameters, nil
}

func parseReturnType(raw any) (vcc.VCCType, error) {
	typeDef, ok := raw.([]any)
	if !ok || len(typeDef) == 0 {
		return "", fmt.Errorf("invalid return type definition")
	}

	typeName, ok := asString(typeDef[0])
	if !ok || typeName == "" {
		return "", fmt.Errorf("missing return type name")
	}

	parsedType, _, err := vcc.ParseVCCType(typeName)
	if err != nil {
		return "", err
	}
	return parsedType, nil
}

func parseParameter(argDef []any) (vcc.Parameter, error) {
	param := vcc.Parameter{}

	typeName, ok := asString(argDef[0])
	if !ok || typeName == "" {
		return param, fmt.Errorf("missing parameter type")
	}

	if strings.EqualFold(typeName, "ENUM") {
		param.Type = vcc.TypeEnum
		param.Enum = &vcc.Enum{
			Values: parseEnumValues(argDef),
		}
	} else {
		paramType, _, err := vcc.ParseVCCType(typeName)
		if err != nil {
			return param, err
		}
		param.Type = paramType
	}

	if len(argDef) > 1 {
		if name, ok := asString(argDef[1]); ok {
			param.Name = name
		}
	}

	if len(argDef) > 2 && argDef[2] != nil {
		if defaultValue, ok := scalarToString(argDef[2]); ok {
			param.DefaultValue = defaultValue
			param.Optional = true
		}
	}

	if len(argDef) > 4 {
		if optional, ok := argDef[4].(bool); ok && optional {
			param.Optional = true
		}
	}

	if param.Enum != nil {
		param.Enum.DefaultValue = param.DefaultValue
	}

	return param, nil
}

func parseEnumValues(argDef []any) []string {
	if len(argDef) <= 3 {
		return []string{}
	}

	valuesRaw, ok := argDef[3].([]any)
	if !ok {
		return []string{}
	}

	values := make([]string, 0, len(valuesRaw))
	for _, raw := range valuesRaw {
		if value, ok := asString(raw); ok {
			values = append(values, value)
		}
	}

	return values
}

func scalarToString(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case json.Number:
		return v.String(), true
	case float64:
		return fmt.Sprintf("%v", v), true
	case bool:
		if v {
			return "true", true
		}
		return "false", true
	default:
		return "", false
	}
}

func asString(value any) (string, bool) {
	str, ok := value.(string)
	if !ok {
		return "", false
	}
	return str, true
}
