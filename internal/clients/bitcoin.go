package clients

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"github.com/jpcummins/satwatch/internal/lib"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type BitcoinClient struct {
	log             zerolog.Logger
	host            string
	user            string
	password        string
	gap             int
	descriptorRegex *regexp.Regexp
	checksumRegex   *regexp.Regexp
}

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      string        `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
	ID     string          `json:"id"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

var (
	descriptorPattern string = `^(?:(?:sortedmulti_a|sortedmulti|multi_a|wpkh|combo|multi|rawtr|raw|addr|wsh|pkh|pk|tr|sh))\(`
	checksumPattern   string = `#([A-Za-z0-9]{8})$`
)

func NewBitcoinClient(host string, user string, password string, gap int) (BitcoinClient, error) {
	log.Info().Msg("initializing Bitcoin client")

	logger := log.With().Str("module", "bitcoin").Str("host", host).Logger()
	logger.Info().Msg("testing connection")

	client := BitcoinClient{
		log:      logger,
		host:     host,
		user:     user,
		password: password,
		gap:      gap,
	}

	descriptorRegex, err := regexp.Compile(descriptorPattern)
	if err != nil {
		logger.Error().Err(err).Msg("Error compiling descriptor regex")
		return client, err
	}
	client.descriptorRegex = descriptorRegex

	checksumRegex, err := regexp.Compile(checksumPattern)
	if err != nil {
		logger.Error().Err(err).Msg("Error compiling checksum regex")
		return client, err
	}
	client.checksumRegex = checksumRegex

	reqBody := rpcRequest{
		JSONRPC: "1.0",
		ID:      "getblockchaininfo",
		Method:  "getblockchaininfo",
		Params:  []interface{}{},
	}

	response, err := client.request(reqBody)
	if err != nil {
		logger.Error().Err(err).Msg("Error testing Bitcoin request")
		return client, err
	}

	logger.Info().Str("result", string(*response)).Msg("success")
	return client, nil
}

func (bc BitcoinClient) request(rpcReq rpcRequest) (*json.RawMessage, error) {
	bodyBytes, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, bc.error(
			"Error marshaling JSON",
			"Internal error",
			err,
		)
	}

	req, err := http.NewRequest("POST", bc.host, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, bc.error(
			"Failed to create request",
			"Internal error",
			err,
		)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(bc.user, bc.password)

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, bc.error(
			"RPC request failed",
			"Internal error",
			err,
		)
	}
	defer resp.Body.Close()

	var rpcResp rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, bc.error(
			"JSON decode error",
			"Internal error",
			err,
		)
	}

	if rpcResp.Error != nil {
		error := errors.New("RPC Error: " + rpcResp.Error.Message)
		return nil, bc.error(
			error.Error(),
			"Internal error",
			error,
		)
	}

	return &rpcResp.Result, nil
}

func (bc BitcoinClient) GetAddressesFromDescriptor(descriptor string) ([]string, error) {
	if !bc.descriptorRegex.MatchString(descriptor) {
		return []string{}, nil
	}

	bc.log.Info().Msg("Descriptor")

	descriptorReq := rpcRequest{
		JSONRPC: "1.0",
		ID:      "getdescriptorinfo",
		Method:  "getdescriptorinfo",
		Params:  []interface{}{descriptor},
	}

	descriptorResponse, err := bc.request(descriptorReq)
	if err != nil {
		return []string{}, bc.error(
			"Error calling getdescriptor info",
			"Internal error",
			err,
		)
	}

	type descriptorInfoResponse struct {
		Descriptor     string `json:"descriptor"`
		Checksum       string `json:"checksum"`
		IsRange        bool   `json:"isrange"`
		IsSolvable     bool   `json:"issolvable"`
		HasPrivateKeys bool   `json:"hasprivatekeys"`
	}

	var descResponse descriptorInfoResponse
	err = json.Unmarshal(*descriptorResponse, &descResponse)
	if err != nil {
		return []string{}, bc.error(
			"Error unmarshaling getdescriptorinfo response",
			"Internal error",
			err,
		)
	}

	if !descResponse.IsSolvable {
		reason := "Unsolvable descriptor"
		return []string{}, bc.error(reason, reason, errors.New(reason))
	}

	if descResponse.HasPrivateKeys {
		reason := "Descriptor contains a private key"
		return []string{}, bc.error(reason, reason, errors.New(reason))
	}

	if !bc.checksumRegex.MatchString(descriptor) {
		descriptor = descriptor + "#" + descResponse.Checksum
	}

	deriveAddressReq := rpcRequest{
		JSONRPC: "1.0",
		ID:      "deriveaddresses",
		Method:  "deriveaddresses",
		Params:  []interface{}{descriptor},
	}

	if descResponse.IsRange {
		deriveAddressReq.Params = append(deriveAddressReq.Params, bc.gap)
	}

	deriveAddressResp, err := bc.request(deriveAddressReq)
	if err != nil {
		return []string{}, bc.error(
			"Error calling deriveaddresses",
			"Internal error",
			err,
		)
	}

	if deriveAddressResp == nil {
		return []string{}, bc.error(
			"No response from deriveaddresses",
			"Internal error",
			err,
		)
	}

	var addresses []string
	err = json.Unmarshal(*deriveAddressResp, &addresses)
	if err != nil {
		return []string{}, bc.error(
			"Error unmarshaling deriveaddresses response",
			"Internal error",
			err,
		)
	}

	return addresses, nil
}

func (bc BitcoinClient) error(internalReason string, displayReason string, err error) error {
	error := lib.Error{
		Err:            err,
		DisplayMessage: displayReason,
	}

	bc.log.Error().Err(err).Str("displayReason", displayReason).Msg(internalReason)
	return error
}
