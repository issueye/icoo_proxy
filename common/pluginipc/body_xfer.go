package pluginipc

// ChooseBodyEncoding picks inline vs raw-followup for a request/response body.
func ChooseBodyEncoding(body []byte, inlineLimit int) (encoding string, attachRaw []byte, bodyField []byte) {
	if inlineLimit <= 0 {
		inlineLimit = DefaultInlineBodyLimit
	}
	if len(body) == 0 {
		return BodyEncodingInline, nil, nil
	}
	if len(body) <= inlineLimit {
		return BodyEncodingInline, nil, body
	}
	return BodyEncodingRawFollowup, body, nil
}

// PrepareProxyRequest fills BodyEncoding / BodyLen / Body fields and returns
// the optional raw attachment for WriteMessage.
func PrepareProxyRequest(req *ProxyRequest, inlineLimit int) (rawAttach []byte) {
	if req == nil {
		return nil
	}
	body := req.Body
	enc, raw, inline := ChooseBodyEncoding(body, inlineLimit)
	req.BodyEncoding = enc
	if enc == BodyEncodingRawFollowup {
		req.Body = nil
		req.BodyLen = len(body)
		return raw
	}
	req.Body = inline
	req.BodyLen = 0
	return nil
}

// PrepareProxyResponse fills encoding fields for a ProxyResponse.
func PrepareProxyResponse(resp *ProxyResponse, inlineLimit int) (rawAttach []byte) {
	if resp == nil {
		return nil
	}
	body := resp.Body
	enc, raw, inline := ChooseBodyEncoding(body, inlineLimit)
	resp.BodyEncoding = enc
	if enc == BodyEncodingRawFollowup {
		resp.Body = nil
		resp.BodyLen = len(body)
		return raw
	}
	resp.Body = inline
	resp.BodyLen = 0
	return nil
}

// ResolveBody returns the effective body from inline field or attached raw body.
func ResolveBody(encoding string, inline []byte, attached []byte) []byte {
	if encoding == BodyEncodingRawFollowup {
		return attached
	}
	return inline
}
