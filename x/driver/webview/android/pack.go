package android

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/lewtec/lewkit/x/driver/webview"
)

const (
	pageOrigin = "https://appassets.androidplatform.net"
	pageHeader = "X-Lewkit-Page"
)

func packResponse(status int, header http.Header, body []byte) []byte {
	if status < 100 {
		status = http.StatusInternalServerError
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%d\n", status)
	for name, values := range header {
		if strings.EqualFold(name, "Content-Length") || strings.EqualFold(name, "Transfer-Encoding") {
			continue
		}
		for _, value := range values {
			value = strings.ReplaceAll(value, "\r", " ")
			value = strings.ReplaceAll(value, "\n", " ")
			buf.WriteString(name)
			buf.WriteString(": ")
			buf.WriteString(value)
			buf.WriteByte('\n')
		}
	}
	buf.WriteByte('\n')
	buf.Write(body)
	return buf.Bytes()
}

func parseHeaderText(text string) http.Header {
	header := make(http.Header)
	for _, line := range strings.Split(text, "\n") {
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		header.Add(name, strings.TrimSpace(value))
	}
	return header
}

func bridgeScript() string {
	name := webview.ScriptName()
	return "(function(){" +
		"if(window.__lewkitPage)return;" +
		"window.__lewkitPage=1;" +
		"window." + name + "={postMessage:function(v){lewkitPage.postMessage(typeof v===\"string\"?v:JSON.stringify(v));}};" +
		"document.addEventListener(\"submit\",function(ev){" +
		"var form=ev.target;" +
		"if(!form||!form.tagName||form.tagName.toUpperCase()!==\"FORM\")return;" +
		"var method=(form.getAttribute(\"method\")||\"GET\").toUpperCase();" +
		"if(method===\"GET\")return;" +
		"ev.preventDefault();" +
		"var body=new URLSearchParams(new FormData(form)).toString();" +
		"lewkitPage.submit(method,form.action||location.href,\"application/x-www-form-urlencoded\",body);" +
		"},true);" +
		"})();"
}

func injectBridge(header http.Header, body []byte) []byte {
	if !htmlBody(header, body) {
		return body
	}
	tag := []byte("<script>" + bridgeScript() + "</script>")
	lower := bytes.ToLower(body)
	if i := bytes.Index(lower, []byte("<head>")); i >= 0 {
		i += len("<head>")
		out := make([]byte, 0, len(body)+len(tag))
		out = append(out, body[:i]...)
		out = append(out, tag...)
		out = append(out, body[i:]...)
		return out
	}
	return append(tag, body...)
}

func htmlBody(header http.Header, body []byte) bool {
	kind := strings.ToLower(header.Get("Content-Type"))
	if strings.Contains(kind, "text/html") || strings.Contains(kind, "application/xhtml") {
		return true
	}
	if kind != "" {
		return false
	}
	return bytes.HasPrefix(bytes.TrimSpace(body), []byte("<"))
}
