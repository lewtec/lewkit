package power

import (
	"context"
	"fmt"
	"net"
)

// Wake sends a Wake-on-LAN magic packet for mac to the subnet broadcast.
// mac is a hardware address net.ParseMAC accepts.
func Wake(ctx context.Context, mac string) error {
	hw, err := net.ParseMAC(mac)
	if err != nil {
		return fmt.Errorf("parse mac %s: %w", mac, err)
	}
	packet, err := magicPacket(hw)
	if err != nil {
		return err
	}
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "udp", "255.255.255.255:9")
	if err != nil {
		return fmt.Errorf("dial udp broadcast: %w", err)
	}
	defer conn.Close()
	if _, err := conn.Write(packet); err != nil {
		return fmt.Errorf("send magic packet: %w", err)
	}
	return nil
}

func magicPacket(hw net.HardwareAddr) ([]byte, error) {
	if len(hw) != 6 {
		return nil, fmt.Errorf("mac length %d", len(hw))
	}
	packet := make([]byte, 6+16*6)
	for i := range 6 {
		packet[i] = 0xFF
	}
	for i := 1; i <= 16; i++ {
		copy(packet[i*6:(i+1)*6], hw)
	}
	return packet, nil
}
