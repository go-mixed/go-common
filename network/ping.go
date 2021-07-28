package network

import (
	"github.com/go-ping/ping"
	"github.com/j-keck/arping"
	"go-common/utils"
	"net"
	"time"
)

func MixPing(ip string, timeout time.Duration, arpIface string) (ok bool, hwAddr string, duration time.Duration) {
	if ok, duration = Ping(ip, timeout); ok {
		return
	}

	return Arping(ip, timeout, arpIface)
}

/** Ping
 * 只能运行在linux上
 * 需要在本机执行后面的语句之后才能ping通 sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"
 */
func Ping(ip string, timeout time.Duration) (ok bool, duration time.Duration) {
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		ok = false
		utils.GetSugaredLogger().Errorf("ping %s fail: %s", ip, err.Error())
		return
	}

	pinger.Count = 3
	pinger.Timeout = timeout
	pinger.Interval = 100 * time.Millisecond
	err = pinger.Run() // Blocks until finished.
	if err != nil {
		ok = false
		utils.GetSugaredLogger().Errorf("ping %s fail: %s", ip, err.Error())
		return
	}

	stats := pinger.Statistics() // get send/receive/duplicate/rtt stats

	if stats.PacketsRecv > 0 {
		ok = true
	} else {
		ok = false
	}

	duration = stats.AvgRtt

	return
}

func Arping(ip string, timeout time.Duration, arpIface string) (ok bool, hwAddr string, duration time.Duration) {
	arping.SetTimeout(timeout)

	_ip := net.ParseIP(ip)

	var err error
	var Addr net.HardwareAddr
	if arpIface != "" {
		Addr, duration, err = arping.PingOverIfaceByName(_ip, arpIface)
	} else {
		Addr, duration, err = arping.Ping(_ip)
	}

	if err != nil {
		ok = false
		utils.GetSugaredLogger().Errorf("arping %s fail: %s", ip, err.Error())
		return
	}

	hwAddr = Addr.String()
	ok = true
	return
}
