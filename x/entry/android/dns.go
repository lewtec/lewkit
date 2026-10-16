//go:build android

package android

import (
	"context"
	"log/slog"
	"net"
	"strings"
	"sync"

	"github.com/lewtec/lewkit/x/ffi/jni"
)

func init() {
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial:     dialDNS,
	}
}

// dialDNS sends the pure-Go lookup to the active network's DNS.
// Android has no resolv.conf. The default lookup then asks [::1]:53,
// and that port refuses the query.
func dialDNS(ctx context.Context, network, address string) (net.Conn, error) {
	var dialer net.Dialer
	if server := dnsServer(); server != "" {
		return dialer.DialContext(ctx, "udp", net.JoinHostPort(server, "53"))
	}
	return dialer.DialContext(ctx, network, address)
}

var (
	dnsMu    sync.Mutex
	dnsHave  bool
	dnsValue string
)

func dnsServer() string {
	dnsMu.Lock()
	defer dnsMu.Unlock()
	if dnsHave {
		return dnsValue
	}
	servers := readDNS()
	if len(servers) == 0 {
		return ""
	}
	dnsHave = true
	dnsValue = servers[0]
	for _, server := range servers {
		if !strings.Contains(server, ":") {
			dnsValue = server
			break
		}
	}
	return dnsValue
}

// readDNS asks ConnectivityManager for the active link's DNS addresses.
func readDNS() []string {
	app, err := jni.Context()
	if err != nil || app == nil {
		return nil
	}
	defer app.Release()
	manager, err := jni.AsRef(app.Call("getSystemService", "connectivity"))
	if err != nil || manager == nil {
		slog.Info("android dns", "err", err)
		return nil
	}
	defer manager.Release()
	network, err := jni.AsRef(manager.Call("getActiveNetwork"))
	if err != nil || network == nil {
		slog.Info("android dns", "err", err)
		return nil
	}
	defer network.Release()
	props, err := jni.AsRef(manager.Call("getLinkProperties", network))
	if err != nil || props == nil {
		slog.Info("android dns", "err", err)
		return nil
	}
	defer props.Release()
	list, err := jni.AsRef(props.Call("getDnsServers"))
	if err != nil || list == nil {
		slog.Info("android dns", "err", err)
		return nil
	}
	defer list.Release()
	count, err := jni.Int(list.Call("size"))
	if err != nil || count < 1 {
		return nil
	}
	var servers []string
	for i := 0; i < count; i++ {
		item, err := jni.AsRef(list.Call("get", i))
		if err != nil || item == nil {
			continue
		}
		host, err := jni.Text(item.Call("getHostAddress"))
		item.Release()
		if err != nil || host == "" || host == "::1" || host == "127.0.0.1" {
			continue
		}
		servers = append(servers, host)
	}
	return servers
}
